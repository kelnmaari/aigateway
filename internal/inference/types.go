package inference

import (
	"context"

	"aigateway/internal/models"
)

// ProviderKind enumerates supported inference providers.
type ProviderKind string

const (
	ProviderVLLM     ProviderKind = "vllm"
	ProviderSGLang   ProviderKind = "sglang"
	ProviderTGI      ProviderKind = "tgi"
	ProviderTEI      ProviderKind = "tei" // Text Embeddings Inference (embedding-only)
	ProviderTRTLLM   ProviderKind = "tensorrt-llm"
	ProviderLlamaCPP ProviderKind = "llama.cpp"
)

// ModelFormat enumerates local model formats.
type ModelFormat string

const (
	FormatHF    ModelFormat = "hf"    // Hugging Face repo (transformers weights)
	FormatGGUF  ModelFormat = "gguf"  // llama.cpp GGUF
	FormatTRT   ModelFormat = "trt"   // TensorRT engine artifacts
	FormatOther ModelFormat = "other" // fallback/custom
)

// Capability is a convenience alias.
type Capability = models.ModelCapability

// ModelSpec describes how a model should be prepared and started.
type ModelSpec struct {
	Alias        string       // user-friendly alias
	Provider     ProviderKind // target provider
	Format       ModelFormat  // model artifact format
	Capabilities []Capability // chat, embeddings, vision
	HFRepo       string       // e.g. meta-llama/Llama-3-8B
	HFRevision   string       // optional revision/commit
	HFFile       string       // filename inside repo
	GGUFURL      string       // direct URL for GGUF if not HF
	ExpectedSHA  string       // optional sha256 for validation
	LocalPath    string       // resolved local path after download

	// GPU selection (e.g., "0", "1", "0,1" for specific GPU(s), empty = all)
	GPUDevice string

	// vLLM-specific
	VLLMTensorParallel int     // --tensor-parallel-size
	VLLMMaxModelLen    int     // --max-model-len
	VLLMGPUUtilization float64 // --gpu-memory-utilization (0..1)
	VLLMExtraArgs           string  // extra CLI args appended to vllm command (e.g. --enable-auto-tool-choice --tool-call-parser hermes)
	VLLMQuantization        string  // --quantization (awq, gptq, squeezellm, fp8)
	VLLMDtype               string  // --dtype (auto, float16, bfloat16, float32)
	VLLMKVCacheDtype        string  // --kv-cache-dtype (auto, fp8)
	VLLMMaxNumSeqs          int     // --max-num-seqs (max concurrent sequences)
	VLLMEnforceEager        bool    // --enforce-eager (disable CUDA graphs)
	VLLMEnablePrefixCaching bool    // --enable-prefix-caching
	VLLMEnableChunkedPrefill bool   // --enable-chunked-prefill
	VLLMSwapSpace           int     // --swap-space (GiB)
	VLLMEnableAutoToolChoice bool  // --enable-auto-tool-choice (enable tool calling)
	VLLMToolCallParser       string // --tool-call-parser (hermes, mistral, llama3_json, llama4_json, deepseek_v3, pythonic, jamba, granite, internlm)
	VLLMChatTemplate         string // --chat-template (path to Jinja template for tool calling)

	// llama.cpp server-specific
	LlamaMainGPU     int    // --main-gpu
	LlamaTensorSplit string // --tensor-split, e.g. "0.5,0.5"
	LlamaNGPULayers  int    // --n-gpu-layers
	LlamaCtxSize     int    // --ctx-size (context window size)
	LlamaNParallel   int    // --parallel (concurrent request slots)
	LlamaFlashAttn   bool   // --flash-attn (enable flash attention)
	LlamaJinja       bool   // --jinja (enable Jinja template processing)
	LlamaCacheReuse  int    // --cache-reuse (0=default, -1=disable for SWA models)
	LlamaExtraArgs   string // extra CLI args appended to llama-server command
	LlamaBatchSize   int    // --batch-size
	LlamaUBatchSize  int    // --ubatch-size
	LlamaCacheTypeK  string // --cache-type-k (f16, q8_0, q4_0)
	LlamaCacheTypeV  string // --cache-type-v (f16, q8_0, q4_0)
	LlamaMlock       bool   // --mlock
	LlamaChatTemplate string // --chat-template (built-in template name, e.g. chatml, mistral, deepseek, llama2)

	// SGLang-specific
	SGLangTensorParallel int     // --tp (tensor parallel)
	SGLangDataParallel   int     // --dp (data parallel)
	SGLangMemFraction    float64 // --mem-fraction-static (0..1)
	SGLangContextLen     int     // --context-length
	SGLangChunkedPrefill   bool    // --chunked-prefill-size (enable chunked prefill)
	SGLangQuantization     string  // --quantization (awq, fp8, gptq, marlin)
	SGLangAttentionBackend string  // --attention-backend (flashinfer, triton, torch_native)
	SGLangExtraArgs        string  // extra CLI args
	SGLangToolCallParser   string  // --tool-call-parser (pythonic, qwen25, qwen, glm47, deepseekv3, minimax-m2)

	// TGI-specific
	TGINumShard          int // --num-shard
	TGIMaxConcurrentReqs int // --max-concurrent-requests
	TGIMaxInputLen       int // --max-input-length
	TGIMaxTotalTokens    int     // --max-total-tokens
	TGIQuantize          string  // --quantize (awq, gptq, eetq, fp8, bitsandbytes, bitsandbytes-nf4)
	TGICudaMemoryFraction float64 // --cuda-memory-fraction (0..1)
	TGIExtraArgs         string  // extra CLI args

	// TEI-specific
	TEIMaxBatchTokens    int    // --max-batch-tokens
	TEIMaxConcurrentReqs int    // --max-concurrent-requests
	TEIPooling           string // --pooling (cls, mean, splade, last-token)
	TEIDtype             string // --dtype (float16, float32)
	TEIExtraArgs         string // extra CLI args
}

// ModelStatus represents container+artifact state.
type ModelStatus string

const (
	StatusPending     ModelStatus = "pending"
	StatusDownloading ModelStatus = "downloading"
	StatusReady       ModelStatus = "ready"
	StatusStarting    ModelStatus = "starting"
	StatusRunning     ModelStatus = "running"
	StatusFailed      ModelStatus = "failed"
)

// ContainerHandle is a minimal runtime handle.
type ContainerHandle struct {
	ID         string
	Provider   ProviderKind
	ModelAlias string
	Endpoint   string // base URL for OpenAI-compatible calls
	HealthURL  string
}

// ContainerRuntime describes how we start/stop provider containers.
type ContainerRuntime interface {
	Start(ctx context.Context, req ContainerStartRequest) (*ContainerHandle, error)
	Stop(ctx context.Context, handleID string) error
	IsRunning(ctx context.Context, handleID string) (bool, error)
	Logs(ctx context.Context, handleID string, tailLines int) (string, error)
}

// ContainerStartRequest holds runtime start parameters.
type ContainerStartRequest struct {
	ModelAlias string
	Provider   ProviderKind
	Image      string
	Command    []string
	Env        map[string]string
	Ports      map[string]int
	Mounts     []VolumeMount
	GPUDevice  string // GPU device(s) to use, e.g., "0", "1", "0,1", empty = all
}

// VolumeMount describes a host->container mount.
type VolumeMount struct {
	HostPath      string
	ContainerPath string
	ReadOnly      bool
}
