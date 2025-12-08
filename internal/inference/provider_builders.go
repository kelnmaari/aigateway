package inference

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"
)

const (
	DefaultVLLMImage     = "vllm/vllm-openai:latest"
	DefaultLlamaImage    = "ghcr.io/ggerganov/llama.cpp:server"
	DefaultSGLangImage   = "arrichm/sglang:latest"
	DefaultTGIImage      = "ghcr.io/huggingface/text-generation-inference:latest"
	DefaultTRTLLMImage   = "nvcr.io/nvidia/tensorrt-llm:latest"
	defaultVLLMPort      = 8000
	defaultLlamaServPort = 8080
	defaultSGLangPort    = 8000
	defaultTGIPort       = 80
	defaultTRTPort       = 8000
)

// BuildVLLMRequest creates a container start request for vLLM openai server.
// Expects spec.LocalPath (preferred) or HFRepo reference.
func BuildVLLMRequest(spec ModelSpec, hfCacheDir string) ContainerStartRequest {
	modelArg := spec.HFRepo
	if spec.LocalPath != "" {
		modelArg = spec.LocalPath
	}

	cmd := []string{
		"--host", "0.0.0.0",
		"--port", fmt.Sprintf("%d", defaultVLLMPort),
		"--model", modelArg,
	}

	if spec.VLLMTensorParallel > 0 {
		cmd = append(cmd, "--tensor-parallel-size", fmt.Sprintf("%d", spec.VLLMTensorParallel))
	}
	if spec.VLLMMaxModelLen > 0 {
		cmd = append(cmd, "--max-model-len", fmt.Sprintf("%d", spec.VLLMMaxModelLen))
	}
	if spec.VLLMGPUUtilization > 0 {
		cmd = append(cmd, "--gpu-memory-utilization", fmt.Sprintf("%.2f", spec.VLLMGPUUtilization))
	}

	return ContainerStartRequest{
		ModelAlias: spec.Alias,
		Provider:   ProviderVLLM,
		Image:      DefaultVLLMImage,
		Command:    cmd,
		Env:        map[string]string{},
		Ports:      map[string]int{"http": defaultVLLMPort},
		Mounts: []VolumeMount{
			{HostPath: hfCacheDir, ContainerPath: "/root/.cache/huggingface", ReadOnly: false},
		},
	}
}

// BuildSGLangRequest creates a container start request for SGLang.
// Supports vision models (Qwen2-VL, LLaVA) and text models with configurable parallelism.
func BuildSGLangRequest(spec ModelSpec, hfCacheDir, hfToken string) ContainerStartRequest {
	modelArg := spec.HFRepo
	if spec.LocalPath != "" {
		modelArg = spec.LocalPath
	}
	cmd := []string{
		"--host", "0.0.0.0",
		"--port", fmt.Sprintf("%d", defaultSGLangPort),
		"--model-path", modelArg,
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

	env := map[string]string{}
	// Only pass HF_TOKEN if model needs to be downloaded (LocalPath empty)
	// Security: isolate token from container when model is already cached
	if hfToken != "" && spec.LocalPath == "" {
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
	}
}

// BuildTGIRequest creates a container start request for TGI (default backend).
// Supports --num-shard for multi-GPU and various performance tuning options.
func BuildTGIRequest(spec ModelSpec, hfCacheDir, hfToken string) ContainerStartRequest {
	modelArg := spec.HFRepo
	if spec.LocalPath != "" {
		modelArg = spec.LocalPath
	}
	env := map[string]string{}
	// Only pass HF_TOKEN if model needs to be downloaded (LocalPath empty)
	// Security: isolate token from container when model is already cached
	if hfToken != "" && spec.LocalPath == "" {
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
	}
}

// BuildTRTLLMRequest creates a container start request for TensorRT-LLM server.
// Assumes prebuilt engine/artifacts available at LocalPath or HF cache.
// Note: HF_TOKEN is NOT passed since TRT-LLM requires pre-converted local engines.
func BuildTRTLLMRequest(spec ModelSpec, enginesDir, _ string) (ContainerStartRequest, error) {
	if spec.LocalPath == "" {
		return ContainerStartRequest{}, fmt.Errorf("trt-llm requires LocalPath to engine/artifacts")
	}
	env := map[string]string{}
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
	if spec.LlamaMainGPU > 0 {
		cmd = append(cmd, "--main-gpu", fmt.Sprintf("%d", spec.LlamaMainGPU))
	}
	if spec.LlamaTensorSplit != "" {
		cmd = append(cmd, "--tensor-split", spec.LlamaTensorSplit)
	}

	return ContainerStartRequest{
		ModelAlias: spec.Alias,
		Provider:   ProviderLlamaCPP,
		Image:      DefaultLlamaImage,
		Command:    cmd,
		Env:        map[string]string{},
		Ports:      map[string]int{"http": defaultLlamaServPort},
		Mounts: []VolumeMount{
			{HostPath: modelDir, ContainerPath: "/models", ReadOnly: true},
		},
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

