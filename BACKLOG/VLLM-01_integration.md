# VLLM-01: vLLM Backend Integration

**Version:** 2.2.3  
**Priority:** HIGH  
**Estimated Time:** 6-8 hours  
**Status:** 📋 Planned  
**Dependencies:** REGISTRY-01, REGISTRY-02

---

## 🎯 Goal

Integrate vLLM as an additional model provider for fast inference of HuggingFace models with GPU acceleration.

---

## 📋 Requirements

### Functional
- vLLM client implementation with OpenAI-compatible API
- Support for multiple models loaded simultaneously
- Multi-GPU configuration and distribution
- Model loading/unloading via API
- Streaming responses compatibility
- Performance metrics collection

### Non-Functional
- < 100ms additional latency vs direct vLLM
- Support для 2+ GPU setup
- Graceful degradation если vLLM unavailable
- Zero config для basic setup

---

## 🏗️ Architecture

### Provider Interface Implementation

```go
// internal/client/vllm/client.go
package vllm

import (
    "context"
    "net/http"
)

type VLLMClient struct {
    baseURL    string
    httpClient *http.Client
    apiKey     string
}

type VLLMConfig struct {
    Enabled    bool     `yaml:"enabled"`
    BaseURL    string   `yaml:"base_url"` // http://localhost:8000
    APIKey     string   `yaml:"api_key"`
    GPUMemory  float64  `yaml:"gpu_memory_utilization"` // 0.9
    MaxModels  int      `yaml:"max_models"` // Max simultaneous models
    TensorParallelSize int `yaml:"tensor_parallel_size"` // 2 for 2 GPUs
}

// Chat completion через vLLM OpenAI-compatible API
func (c *VLLMClient) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)

// Streaming chat completion
func (c *VLLMClient) ChatCompletionStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)

// List available models в vLLM
func (c *VLLMClient) ListModels(ctx context.Context) ([]Model, error)

// Load model into vLLM
func (c *VLLMClient) LoadModel(ctx context.Context, modelName string, opts LoadOptions) error

// Unload model from vLLM
func (c *VLLMClient) UnloadModel(ctx context.Context, modelName string) error

// Health check
func (c *VLLMClient) Health(ctx context.Context) (*HealthStatus, error)
```

### Request Conversion

vLLM uses OpenAI-compatible API, minimal conversion needed:

```go
// Convert AIGateway ChatRequest → vLLM format
func convertToVLLMRequest(req models.ChatCompletionRequest) vllmRequest {
    return vllmRequest{
        Model:       req.Model,
        Messages:    req.Messages,
        Temperature: req.Temperature,
        MaxTokens:   req.MaxTokens,
        Stream:      req.Stream,
        // vLLM specific parameters
        TopP:             req.TopP,
        FrequencyPenalty: req.FrequencyPenalty,
        PresencePenalty:  req.PresencePenalty,
    }
}
```

### Model Registry Integration

```go
// Register vLLM models в registry при старте
func (c *VLLMClient) RegisterModels(registry *models.ModelRegistry) error {
    models, err := c.ListModels(ctx)
    if err != nil {
        return err
    }
    
    for _, model := range models {
        registry.RegisterModel(models.ModelInfo{
            ID:           model.ID,
            Name:         model.ID,
            Source:       models.SourceVLLM,
            Endpoint:     c.baseURL,
            Capabilities: []string{"chat", "streaming"},
            RequiresGPU:  true,
            Provider:     "vllm",
        })
    }
    
    return nil
}
```

---

## 🔧 Implementation Plan

### Phase 1: vLLM Client (2-3h)

```go
// internal/client/vllm/client.go
type VLLMClient struct {
    baseURL    string
    httpClient *http.Client
}

func NewVLLMClient(config VLLMConfig) *VLLMClient
func (c *VLLMClient) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
func (c *VLLMClient) ChatCompletionStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
func (c *VLLMClient) ListModels(ctx context.Context) ([]Model, error)
```

### Phase 2: Configuration (1h)

```yaml
# configs/dev.yaml
providers:
  ollama:
    enabled: true
    base_url: http://localhost:11434
    
  vllm:
    enabled: true
    base_url: http://localhost:8000
    api_key: ""
    gpu_memory_utilization: 0.9
    max_models: 2
    tensor_parallel_size: 2  # For 2x GPU setup
    trust_remote_code: false
```

### Phase 3: Router Integration (1-2h)

```go
// internal/api/handlers/chat.go
func (h *ChatHandler) HandleChat(c *gin.Context) {
    modelName := extractModelName(c)
    
    // Get model info from registry
    modelInfo, err := h.registry.GetModel(modelName)
    if err != nil {
        return handleError(c, err)
    }
    
    // Route to appropriate provider
    switch modelInfo.Source {
    case models.SourceOllama:
        return h.ollamaClient.ChatCompletion(ctx, req)
    case models.SourceVLLM:
        return h.vllmClient.ChatCompletion(ctx, req)
    default:
        return handleError(c, ErrUnknownProvider)
    }
}
```

### Phase 4: Health Checks (1h)

```go
// internal/health/vllm_checker.go
type VLLMHealthChecker struct {
    client *vllm.VLLMClient
}

func (hc *VLLMHealthChecker) Check(ctx context.Context) (*HealthStatus, error) {
    status, err := hc.client.Health(ctx)
    return &HealthStatus{
        Provider: "vllm",
        Status:   status.Status,
        Models:   status.LoadedModels,
        GPUInfo:  status.GPUMemory,
    }, err
}
```

### Phase 5: Docker Compose (1h)

```yaml
# docker-compose.yml
services:
  aigateway:
    # ... existing config
    depends_on:
      - ollama
      - vllm
      
  vllm:
    image: vllm/vllm-openai:latest
    ports:
      - "8000:8000"
    volumes:
      - vllm-models:/root/.cache/huggingface
    environment:
      - CUDA_VISIBLE_DEVICES=0,1  # Both GPUs
    command: >
      --model meta-llama/Llama-2-7b-chat-hf
      --tensor-parallel-size 2
      --gpu-memory-utilization 0.9
      --host 0.0.0.0
      --port 8000
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 2
              capabilities: [gpu]
```

### Phase 6: Performance Testing (1-2h)

```go
// internal/client/vllm/client_test.go
func BenchmarkVLLMChatCompletion(b *testing.B) {
    client := NewVLLMClient(testConfig)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := client.ChatCompletion(ctx, testRequest)
        if err != nil {
            b.Fatal(err)
        }
    }
}

// Compare vLLM vs Ollama performance
func BenchmarkProviderComparison(b *testing.B) {
    // Test same model via different providers
}
```

---

## 📚 Documentation: vLLM Installation Guide

Create **docs/VLLM_INSTALLATION.md**:

```markdown
# vLLM Installation Guide для Linux с 2x GPU

## Prerequisites
- Ubuntu 20.04+ / Debian 11+
- Python 3.8-3.11
- CUDA 11.8+ или 12.1+
- 2x NVIDIA GPU (3080, 4090, A100, etc)
- 32GB+ RAM recommended

## Step 1: Verify CUDA

\`\`\`bash
nvidia-smi  # Должны видеть обе GPU
nvcc --version  # CUDA version
\`\`\`

## Step 2: Install vLLM

\`\`\`bash
# Создать Python environment
python3 -m venv vllm-env
source vllm-env/bin/activate

# Install vLLM с CUDA support
pip install vllm

# Verify installation
python -c "import vllm; print(vllm.__version__)"
\`\`\`

## Step 3: Download Model

\`\`\`bash
# HuggingFace Hub login (если требуется)
pip install huggingface-hub
huggingface-cli login

# Models будут auto-downloaded при первом запуске
# или pre-download:
python -c "from transformers import AutoModelForCausalLM; \
           AutoModelForCausalLM.from_pretrained('meta-llama/Llama-2-7b-chat-hf')"
\`\`\`

## Step 4: Launch vLLM Server (2x GPU)

\`\`\`bash
# Terminal 1: vLLM Server
CUDA_VISIBLE_DEVICES=0,1 python -m vllm.entrypoints.openai.api_server \
    --model meta-llama/Llama-2-7b-chat-hf \
    --tensor-parallel-size 2 \
    --gpu-memory-utilization 0.9 \
    --host 0.0.0.0 \
    --port 8000

# Logs should show:
# INFO: Using 2 GPUs for tensor parallelism
# INFO: Loaded model meta-llama/Llama-2-7b-chat-hf
# INFO: Server started at http://0.0.0.0:8000
\`\`\`

## Step 5: Test vLLM

\`\`\`bash
# Test OpenAI-compatible API
curl http://localhost:8000/v1/models

curl http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "meta-llama/Llama-2-7b-chat-hf",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
\`\`\`

## Step 6: Configure AIGateway

\`\`\`yaml
# configs/dev.yaml
providers:
  vllm:
    enabled: true
    base_url: "http://localhost:8000"
    tensor_parallel_size: 2
    gpu_memory_utilization: 0.9
\`\`\`

## Recommended Models для 3080 (10GB VRAM each)

| Model | Size | VRAM Usage (2x GPU) | Performance |
|-------|------|---------------------|-------------|
| Llama-2-7B-chat | 7B | ~8GB | ⚡⚡⚡⚡ |
| Mistral-7B-Instruct | 7B | ~8GB | ⚡⚡⚡⚡⚡ |
| Llama-2-13B-chat | 13B | ~16GB | ⚡⚡⚡ |
| CodeLlama-13B | 13B | ~16GB | ⚡⚡⚡ |

## Troubleshooting

### OutOfMemoryError
\`\`\`bash
# Reduce GPU memory utilization
--gpu-memory-utilization 0.85  # instead of 0.9
\`\`\`

### Model not found
\`\`\`bash
# Check HuggingFace cache
ls ~/.cache/huggingface/hub/
\`\`\`

### CUDA errors
\`\`\`bash
# Verify CUDA compatibility
python -c "import torch; print(torch.cuda.is_available())"
\`\`\`
```

---

## ✅ Acceptance Criteria

- [ ] vLLM client успешно подключается к vLLM server
- [ ] Chat completions работают через vLLM provider
- [ ] Streaming responses functional
- [ ] Models auto-register в Model Registry
- [ ] Health checks показывают vLLM status
- [ ] Docker Compose включает vLLM service
- [ ] Documentation guide протестирован на Ubuntu 22.04
- [ ] Performance benchmarks completed
- [ ] 2x GPU setup verified

---

## 🧪 Testing

```bash
# Unit tests
go test ./internal/client/vllm/...

# Integration test с vLLM server
go test ./internal/client/vllm/... -tags=integration

# Load test
go test -bench=. ./internal/client/vllm/...
```

---

## 📈 Success Metrics

- vLLM requests < 100ms additional latency
- Support для 3+ models simultaneously
- 100% OpenAI API compatibility
- Zero-config startup with Docker Compose

---

**Next Steps после VLLM-01:**
- HUGGINGFACE-01: HuggingFace Hub model downloader
- VLLM-02: Model hot-swap без downtime
- VLLM-03: Advanced vLLM features (LoRA adapters, quantization)

