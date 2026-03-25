package inference

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultVLLMImage     = "vllm/vllm-openai:latest"
	DefaultLlamaImage    = "ghcr.io/ggml-org/llama.cpp:server-cuda"
	DefaultSGLangImage   = "lmsysorg/sglang:latest"
	DefaultTGIImage      = "ghcr.io/huggingface/text-generation-inference:latest"
	DefaultTEIImage      = "ghcr.io/huggingface/text-embeddings-inference:89-1.8" // Use 1.7 for stability, GPU: :1.7
	DefaultTRTLLMImage   = "nvcr.io/nvidia/tritonserver:24.12-trtllm-python-py3"
	defaultVLLMPort      = 8000
	defaultLlamaServPort = 8080
	defaultSGLangPort    = 8000
	defaultTGIPort       = 80
	defaultTEIPort       = 80
	defaultTRTPort       = 8000
)

// BuildVLLMRequest creates a container start request for vLLM openai server.
// Expects spec.LocalPath (preferred) or HFRepo reference.
func BuildVLLMRequest(spec ModelSpec, hfCacheDir, hfToken string) ContainerStartRequest {
	modelArg := spec.HFRepo
	useLocalModel := false

	if spec.LocalPath != "" {
		// Explicit local path provided
		modelArg = spec.LocalPath
		useLocalModel = true
	} else if spec.HFRepo != "" {
		// Check if model is already downloaded in cache directory
		localModelPath := filepath.Join(hfCacheDir, spec.HFRepo)
		if info, err := os.Stat(localModelPath); err == nil && info.IsDir() {
			// Model exists locally, use container path
			modelArg = "/root/.cache/huggingface/" + spec.HFRepo
			useLocalModel = true
		}
	}

	cmd := []string{
		"--host", "0.0.0.0",
		"--port", fmt.Sprintf("%d", defaultVLLMPort),
		"--model", modelArg,
		"--served-model-name", spec.Alias, // Expose model under alias for API compatibility
		"--trust-remote-code", // Allow custom model code from HuggingFace
	}

	if spec.VLLMTensorParallel > 0 {
		cmd = append(cmd, "--tensor-parallel-size", fmt.Sprintf("%d", spec.VLLMTensorParallel))
	}
	// Set max_model_len - use specified value or default to 32768 to avoid OOM with large context models
	maxModelLen := spec.VLLMMaxModelLen
	if maxModelLen == 0 {
		maxModelLen = 32768 // Reasonable default for most use cases
	}
	cmd = append(cmd, "--max-model-len", fmt.Sprintf("%d", maxModelLen))

	if spec.VLLMGPUUtilization > 0 {
		cmd = append(cmd, "--gpu-memory-utilization", fmt.Sprintf("%.2f", spec.VLLMGPUUtilization))
	}
	if spec.VLLMQuantization != "" {
		cmd = append(cmd, "--quantization", spec.VLLMQuantization)
	}
	if spec.VLLMDtype != "" && spec.VLLMDtype != "auto" {
		cmd = append(cmd, "--dtype", spec.VLLMDtype)
	}
	if spec.VLLMKVCacheDtype != "" && spec.VLLMKVCacheDtype != "auto" {
		cmd = append(cmd, "--kv-cache-dtype", spec.VLLMKVCacheDtype)
	}
	if spec.VLLMMaxNumSeqs > 0 {
		cmd = append(cmd, "--max-num-seqs", fmt.Sprintf("%d", spec.VLLMMaxNumSeqs))
	}
	if spec.VLLMEnforceEager {
		cmd = append(cmd, "--enforce-eager")
	}
	if spec.VLLMEnablePrefixCaching {
		cmd = append(cmd, "--enable-prefix-caching")
	}
	if spec.VLLMEnableChunkedPrefill {
		cmd = append(cmd, "--enable-chunked-prefill")
	}
	if spec.VLLMSwapSpace > 0 {
		cmd = append(cmd, "--swap-space", fmt.Sprintf("%d", spec.VLLMSwapSpace))
	}

	// Extra args: split by whitespace and append as raw CLI args
	// e.g. "--enable-auto-tool-choice --tool-call-parser hermes"
	if spec.VLLMExtraArgs != "" {
		cmd = append(cmd, strings.Fields(spec.VLLMExtraArgs)...)
	}

	env := map[string]string{
		"CUDA_DEVICE_ORDER": "PCI_BUS_ID", // Ensure consistent GPU ordering
		// Enable verbose logging for debugging
		"VLLM_LOGGING_LEVEL":        "DEBUG",
		"TRANSFORMERS_VERBOSITY":    "info",
		"HF_HUB_ENABLE_HF_TRANSFER": "1", // Faster downloads
	}
	// Pass HF token only if downloading from HuggingFace
	if hfToken != "" && !useLocalModel {
		env["HF_TOKEN"] = hfToken
	}
	return ContainerStartRequest{
		ModelAlias: spec.Alias,
		Provider:   ProviderVLLM,
		Image:      DefaultVLLMImage,
		Command:    cmd,
		Env:        env,
		Ports:      map[string]int{"http": defaultVLLMPort},
		Mounts: []VolumeMount{
			{HostPath: hfCacheDir, ContainerPath: "/root/.cache/huggingface", ReadOnly: false},
		},
		GPUDevice: spec.GPUDevice,
	}
}

// BuildSGLangRequest creates a container start request for SGLang.
// Supports vision models (Qwen2-VL, LLaVA) and text models with configurable parallelism.
func BuildSGLangRequest(spec ModelSpec, hfCacheDir, hfToken string) ContainerStartRequest {
	modelArg := spec.HFRepo
	useLocalModel := false

	if spec.LocalPath != "" {
		// Explicit local path provided
		modelArg = spec.LocalPath
		useLocalModel = true
	} else if spec.HFRepo != "" {
		// Check if model is already downloaded in cache directory
		localModelPath := filepath.Join(hfCacheDir, spec.HFRepo)
		if info, err := os.Stat(localModelPath); err == nil && info.IsDir() {
			// Model exists locally, use container path
			modelArg = "/root/.cache/huggingface/" + spec.HFRepo
			useLocalModel = true
		}
	}

	// SGLang uses "python -m sglang.launch_server" as entrypoint in the container
	// Arguments: --model for HF model ID, --host, --port
	cmd := []string{
		"python", "-m", "sglang.launch_server",
		"--model", modelArg,
		"--host", "0.0.0.0",
		"--port", fmt.Sprintf("%d", defaultSGLangPort),
		"--trust-remote-code", // Allow custom model code from HuggingFace
	}

	// Tensor parallelism (multi-GPU)
	if spec.SGLangTensorParallel > 0 {
		cmd = append(cmd, "--tp", fmt.Sprintf("%d", spec.SGLangTensorParallel))
	}

	// Data parallelism
	if spec.SGLangDataParallel > 0 {
		cmd = append(cmd, "--dp", fmt.Sprintf("%d", spec.SGLangDataParallel))
	}

	// Memory fraction
	if spec.SGLangMemFraction > 0 {
		cmd = append(cmd, "--mem-fraction-static", fmt.Sprintf("%.2f", spec.SGLangMemFraction))
	}

	// Context length
	if spec.SGLangContextLen > 0 {
		cmd = append(cmd, "--context-length", fmt.Sprintf("%d", spec.SGLangContextLen))
	}

	// Chunked prefill for long context
	if spec.SGLangChunkedPrefill {
		cmd = append(cmd, "--chunked-prefill-size", "8192")
	}
	if spec.SGLangQuantization != "" {
		cmd = append(cmd, "--quantization", spec.SGLangQuantization)
	}
	if spec.SGLangAttentionBackend != "" {
		cmd = append(cmd, "--attention-backend", spec.SGLangAttentionBackend)
	}
	// SGLang extra args
	if spec.SGLangExtraArgs != "" {
		for _, arg := range strings.Fields(spec.SGLangExtraArgs) {
			cmd = append(cmd, arg)
		}
	}

	env := map[string]string{
		"CUDA_DEVICE_ORDER": "PCI_BUS_ID",
	}
	// Only pass HF_TOKEN if model needs to be downloaded
	if hfToken != "" && !useLocalModel {
		env["HF_TOKEN"] = hfToken
	}
	return ContainerStartRequest{
		ModelAlias: spec.Alias,
		Provider:   ProviderSGLang,
		Image:      DefaultSGLangImage,
		Command:    cmd,
		Env:        env,
		Ports:      map[string]int{"http": defaultSGLangPort},
		Mounts: []VolumeMount{
			{HostPath: hfCacheDir, ContainerPath: "/root/.cache/huggingface", ReadOnly: false},
		},
		GPUDevice: spec.GPUDevice,
	}
}

// BuildTGIRequest creates a container start request for TGI (default backend).
// Supports --num-shard for multi-GPU and various performance tuning options.
func BuildTGIRequest(spec ModelSpec, hfCacheDir, hfToken string) ContainerStartRequest {
	modelArg := spec.HFRepo
	useLocalModel := false

	if spec.LocalPath != "" {
		// Explicit local path provided
		modelArg = spec.LocalPath
		useLocalModel = true
	} else if spec.HFRepo != "" {
		// Check if model is already downloaded in cache directory
		localModelPath := filepath.Join(hfCacheDir, spec.HFRepo)
		if info, err := os.Stat(localModelPath); err == nil && info.IsDir() {
			// Model exists locally, use container path (TGI mounts to /data)
			modelArg = "/data/" + spec.HFRepo
			useLocalModel = true
		}
	}

	env := map[string]string{
		"CUDA_DEVICE_ORDER": "PCI_BUS_ID",
	}
	// Only pass HF_TOKEN if model needs to be downloaded
	if hfToken != "" && !useLocalModel {
		env["HUGGINGFACE_HUB_TOKEN"] = hfToken
	}
	cmd := []string{
		"--model-id", modelArg,
	}

	// Multi-GPU sharding
	if spec.TGINumShard > 0 {
		cmd = append(cmd, "--num-shard", fmt.Sprintf("%d", spec.TGINumShard))
	}

	// Max concurrent requests
	if spec.TGIMaxConcurrentReqs > 0 {
		cmd = append(cmd, "--max-concurrent-requests", fmt.Sprintf("%d", spec.TGIMaxConcurrentReqs))
	}

	// Max input length
	if spec.TGIMaxInputLen > 0 {
		cmd = append(cmd, "--max-input-length", fmt.Sprintf("%d", spec.TGIMaxInputLen))
	}

	// Max total tokens (input + output)
	if spec.TGIMaxTotalTokens > 0 {
		cmd = append(cmd, "--max-total-tokens", fmt.Sprintf("%d", spec.TGIMaxTotalTokens))
	}
	if spec.TGIQuantize != "" {
		cmd = append(cmd, "--quantize", spec.TGIQuantize)
	}
	if spec.TGICudaMemoryFraction > 0 && spec.TGICudaMemoryFraction < 1.0 {
		cmd = append(cmd, "--cuda-memory-fraction", fmt.Sprintf("%.2f", spec.TGICudaMemoryFraction))
	}
	// TGI extra args
	if spec.TGIExtraArgs != "" {
		for _, arg := range strings.Fields(spec.TGIExtraArgs) {
			cmd = append(cmd, arg)
		}
	}

	return ContainerStartRequest{
		ModelAlias: spec.Alias,
		Provider:   ProviderTGI,
		Image:      DefaultTGIImage,
		Command:    cmd,
		Env:        env,
		Ports:      map[string]int{"http": defaultTGIPort},
		Mounts: []VolumeMount{
			{HostPath: hfCacheDir, ContainerPath: "/data", ReadOnly: false},
		},
		GPUDevice: spec.GPUDevice,
	}
}

// BuildTEIRequest creates a container start request for Text Embeddings Inference.
// TEI is optimized for embedding models (sentence-transformers, nomic, etc).
// API: POST /embed with {inputs: ["text"]} -> {embeddings: [[...]]}
func BuildTEIRequest(spec ModelSpec, hfCacheDir, hfToken string) ContainerStartRequest {
	modelArg := spec.HFRepo
	useLocalModel := false

	if spec.LocalPath != "" {
		// Explicit local path provided
		modelArg = spec.LocalPath
		useLocalModel = true
	} else if spec.HFRepo != "" {
		// Check if model is already downloaded in cache directory
		localModelPath := filepath.Join(hfCacheDir, spec.HFRepo)
		if info, err := os.Stat(localModelPath); err == nil && info.IsDir() {
			// Model exists locally, use container path
			modelArg = "/data/" + spec.HFRepo
			useLocalModel = true
		}
	}

	env := map[string]string{
		"CUDA_DEVICE_ORDER": "PCI_BUS_ID",
		"HF_HOME":           "/data", // Tell TEI to use mounted cache directory
	}
	// Pass HF_TOKEN only if downloading from HuggingFace
	if hfToken != "" && !useLocalModel {
		env["HUGGINGFACE_HUB_TOKEN"] = hfToken
		env["HF_TOKEN"] = hfToken // Some versions use HF_TOKEN
	}

	// TEI command arguments
	cmd := []string{
		"--model-id", modelArg,
		"--port", fmt.Sprintf("%d", defaultTEIPort),
	}
	if spec.TEIMaxBatchTokens > 0 {
		cmd = append(cmd, "--max-batch-tokens", fmt.Sprintf("%d", spec.TEIMaxBatchTokens))
	}
	if spec.TEIMaxConcurrentReqs > 0 {
		cmd = append(cmd, "--max-concurrent-requests", fmt.Sprintf("%d", spec.TEIMaxConcurrentReqs))
	}
	if spec.TEIPooling != "" {
		cmd = append(cmd, "--pooling", spec.TEIPooling)
	}
	if spec.TEIDtype != "" {
		cmd = append(cmd, "--dtype", spec.TEIDtype)
	}
	// TEI extra args
	if spec.TEIExtraArgs != "" {
		for _, arg := range strings.Fields(spec.TEIExtraArgs) {
			cmd = append(cmd, arg)
		}
	}

	// Determine if GPU is available - use GPU image variant
	image := DefaultTEIImage
	if spec.GPUDevice != "" {
		// Use GPU variant for better performance
		image = "ghcr.io/huggingface/text-embeddings-inference:89-1.8"
	}

	return ContainerStartRequest{
		ModelAlias: spec.Alias,
		Provider:   ProviderTEI,
		Image:      image,
		Command:    cmd,
		Env:        env,
		Ports:      map[string]int{"http": defaultTEIPort},
		Mounts: []VolumeMount{
			{HostPath: hfCacheDir, ContainerPath: "/data", ReadOnly: false},
		},
		GPUDevice: spec.GPUDevice,
	}
}

// BuildTRTLLMRequest creates a container start request for TensorRT-LLM server.
// Assumes prebuilt engine/artifacts available at LocalPath or HF cache.
// Note: HF_TOKEN is NOT passed since TRT-LLM requires pre-converted local engines.
func BuildTRTLLMRequest(spec ModelSpec, enginesDir, _ string) (ContainerStartRequest, error) {
	if spec.LocalPath == "" {
		return ContainerStartRequest{}, fmt.Errorf("trt-llm requires LocalPath to engine/artifacts")
	}
	env := map[string]string{
		"CUDA_DEVICE_ORDER": "PCI_BUS_ID",
	}
	// Placeholder command: rely on image entrypoint; users may need to override via HF repo config.
	cmd := []string{}
	return ContainerStartRequest{
		ModelAlias: spec.Alias,
		Provider:   ProviderTRTLLM,
		Image:      DefaultTRTLLMImage,
		Command:    cmd,
		Env:        env,
		Ports:      map[string]int{"http": defaultTRTPort},
		Mounts: []VolumeMount{
			{HostPath: filepath.Dir(spec.LocalPath), ContainerPath: "/engines", ReadOnly: false},
			{HostPath: enginesDir, ContainerPath: "/data", ReadOnly: false},
		},
		GPUDevice: spec.GPUDevice,
	}, nil
}

// BuildLlamaCPPRequest creates a container start request for llama.cpp server (GGUF).
func BuildLlamaCPPRequest(spec ModelSpec) (ContainerStartRequest, error) {
	if spec.LocalPath == "" {
		return ContainerStartRequest{}, fmt.Errorf("llama.cpp requires LocalPath")
	}

	modelDir := filepath.Dir(spec.LocalPath)
	modelFile := filepath.Base(spec.LocalPath)

	cmd := []string{
		"--host", "0.0.0.0",
		"--port", fmt.Sprintf("%d", defaultLlamaServPort),
		"--model", "/models/" + modelFile,
	}

	if spec.LlamaNGPULayers > 0 {
		cmd = append(cmd, "--n-gpu-layers", fmt.Sprintf("%d", spec.LlamaNGPULayers))
	}
	if spec.LlamaCtxSize > 0 {
		cmd = append(cmd, "--ctx-size", fmt.Sprintf("%d", spec.LlamaCtxSize))
	}
	if spec.LlamaNParallel > 0 {
		cmd = append(cmd, "--parallel", fmt.Sprintf("%d", spec.LlamaNParallel))
	}
	// Flash attention: explicit on/off (default in llama.cpp is 'auto')
	if spec.LlamaFlashAttn {
		cmd = append(cmd, "--flash-attn", "on")
	} else {
		cmd = append(cmd, "--flash-attn", "off")
	}
	if spec.LlamaMainGPU > 0 {
		cmd = append(cmd, "--main-gpu", fmt.Sprintf("%d", spec.LlamaMainGPU))
	}
	if spec.LlamaTensorSplit != "" {
		cmd = append(cmd, "--tensor-split", spec.LlamaTensorSplit)
	}
	if spec.LlamaJinja {
		cmd = append(cmd, "--jinja")
	}
	// Cache reuse: 0=don't pass (llama.cpp default 256), -1=disable, >0=set value
	if spec.LlamaCacheReuse != 0 {
		cmd = append(cmd, "--cache-reuse", fmt.Sprintf("%d", spec.LlamaCacheReuse))
	}
	if spec.LlamaBatchSize > 0 {
		cmd = append(cmd, "--batch-size", fmt.Sprintf("%d", spec.LlamaBatchSize))
	}
	if spec.LlamaUBatchSize > 0 {
		cmd = append(cmd, "--ubatch-size", fmt.Sprintf("%d", spec.LlamaUBatchSize))
	}
	if spec.LlamaCacheTypeK != "" && spec.LlamaCacheTypeK != "f16" {
		cmd = append(cmd, "--cache-type-k", spec.LlamaCacheTypeK)
	}
	if spec.LlamaCacheTypeV != "" && spec.LlamaCacheTypeV != "f16" {
		cmd = append(cmd, "--cache-type-v", spec.LlamaCacheTypeV)
	}
	if spec.LlamaMlock {
		cmd = append(cmd, "--mlock")
	}
	// Extra args: split by whitespace and append as raw CLI args
	if spec.LlamaExtraArgs != "" {
		cmd = append(cmd, strings.Fields(spec.LlamaExtraArgs)...)
	}

	return ContainerStartRequest{
		ModelAlias: spec.Alias,
		Provider:   ProviderLlamaCPP,
		Image:      DefaultLlamaImage,
		Command:    cmd,
		Env:        map[string]string{"CUDA_DEVICE_ORDER": "PCI_BUS_ID"},
		Ports:      map[string]int{"http": defaultLlamaServPort},
		Mounts: []VolumeMount{
			{HostPath: modelDir, ContainerPath: "/models", ReadOnly: true},
		},
		GPUDevice: spec.GPUDevice,
	}, nil
}

// providerHealthURL returns default health endpoint for provider.
func providerHealthURL(provider ProviderKind, base string) string {
	switch provider {
	case ProviderVLLM:
		return fmt.Sprintf("%s/health", base)
	case ProviderLlamaCPP:
		return fmt.Sprintf("%s/health", base)
	case ProviderSGLang:
		return fmt.Sprintf("%s/health", base)
	case ProviderTGI:
		return fmt.Sprintf("%s/health", base)
	default:
		return fmt.Sprintf("%s/health", base)
	}
}

// providerMetricsURL returns metrics endpoint for provider (if supported).
func providerMetricsURL(provider ProviderKind, base string) string {
	switch provider {
	case ProviderVLLM:
		return fmt.Sprintf("%s/metrics", base) // Prometheus metrics
	case ProviderTGI:
		return fmt.Sprintf("%s/metrics", base)
	case ProviderSGLang:
		return fmt.Sprintf("%s/metrics", base)
	default:
		return "" // llama.cpp server has no /metrics by default
	}
}

// HealthCheckHTTP performs a simple GET health check.
func HealthCheckHTTP(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}
	return nil
}

// fetchMetricsHTTP retrieves raw metrics (Prometheus text format) from url.
func fetchMetricsHTTP(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("metrics fetch failed: status %d", resp.StatusCode)
	}
	buf := make([]byte, 256*1024)
	n, _ := resp.Body.Read(buf)
	return string(buf[:n]), nil
}
