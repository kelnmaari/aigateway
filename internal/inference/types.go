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
	Alias        string             // user-friendly alias
	Provider     ProviderKind       // target provider
	Format       ModelFormat        // model artifact format
	Capabilities []Capability       // chat, embeddings, vision
	HFRepo       string             // e.g. meta-llama/Llama-3-8B
	HFRevision   string             // optional revision/commit
	HFFile       string             // filename inside repo
	GGUFURL      string             // direct URL for GGUF if not HF
	ExpectedSHA  string             // optional sha256 for validation
	LocalPath    string             // resolved local path after download

	// GPU selection (e.g., "0", "1", "0,1" for specific GPU(s), empty = all)
	GPUDevice string

	// vLLM-specific
	VLLMTensorParallel int     // --tensor-parallel-size
	VLLMMaxModelLen    int     // --max-model-len
	VLLMGPUUtilization float64 // --gpu-memory-utilization (0..1)

	// llama.cpp server-specific
	LlamaMainGPU     int    // --main-gpu
	LlamaTensorSplit string // --tensor-split, e.g. "0.5,0.5"
	LlamaNGPULayers  int    // --n-gpu-layers

	// SGLang-specific
	SGLangTensorParallel int    // --tp (tensor parallel)
	SGLangDataParallel   int    // --dp (data parallel)
	SGLangMemFraction    float64 // --mem-fraction-static (0..1)
	SGLangContextLen     int    // --context-length
	SGLangChunkedPrefill bool   // --chunked-prefill-size (enable chunked prefill)

	// TGI-specific
	TGINumShard           int // --num-shard
	TGIMaxConcurrentReqs  int // --max-concurrent-requests
	TGIMaxInputLen        int // --max-input-length
	TGIMaxTotalTokens     int // --max-total-tokens
}

// ModelStatus represents container+artifact state.
type ModelStatus string

const (
	StatusPending   ModelStatus = "pending"
	StatusDownloading ModelStatus = "downloading"
	StatusReady     ModelStatus = "ready"
	StatusStarting  ModelStatus = "starting"
	StatusRunning   ModelStatus = "running"
	StatusFailed    ModelStatus = "failed"
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

