# yzma Integration Guide

## Status: ✅ IMPLEMENTED (v3.0.0-dev)

The yzma integration in v3.0.0-dev is **fully functional** and provides local LLM inference using llama.cpp without external services like Ollama.

## What is yzma?

`yzma` is a Go library that provides local inference for Vision Language Models (VLMs), Large Language Models (LLMs), Small Language Models (SLMs), and Tiny Language Models (TLMs) using `llama.cpp` bindings without CGo.

- GitHub: https://github.com/hybridgroup/yzma
- Uses llama.cpp for high-performance inference
- Supports GGUF model format
- Hardware acceleration on Linux, macOS, Windows

## Current Implementation

### ✅ Implemented (v3.0.0-dev):
- ✅ Full yzma client wrapper (`internal/yzma/client.go`)
- ✅ llama.cpp library loading and initialization
- ✅ GGUF model loading and management
- ✅ Model context caching
- ✅ Text generation (non-streaming)
- ✅ Streaming generation with SSE
- ✅ OpenAI-compatible API handler (`internal/api/handlers/yzma_handler.go`)
- ✅ Chat template support
- ✅ Sampler chain (Temperature, Top-K, Top-P, Min-P)
- ✅ Stop sequences
- ✅ Context cancellation
- ✅ Token counting and stats
- ✅ Configuration system
- ✅ Graceful shutdown

### 🔄 Pending:
- ⏳ Router integration (next step)
- ⏳ VLM support (multimodal)
- ⏳ Batch inference
- ⏳ GPU acceleration configuration

## Requirements for Full Integration

### 1. llama.cpp Shared Libraries

Download pre-built libraries for your platform:

```bash
# Linux (CPU only)
wget https://github.com/ggml-org/llama.cpp/releases/download/b4415/libllama-b4415-bin-ubuntu-x64.zip

# Linux (CUDA)
wget https://github.com/ggml-org/llama.cpp/releases/download/b4415/libllama-b4415-bin-ubuntu-x64-cuda-cu12.2.0.zip

# macOS (Apple Silicon)
wget https://github.com/ggml-org/llama.cpp/releases/download/b4415/libllama-b4415-bin-macos-arm64.zip

# Windows (CPU)
wget https://github.com/ggml-org/llama.cpp/releases/download/b4415/libllama-b4415-bin-win-llama-cu12.2.0-x64.zip
```

Extract and set environment variable:

```bash
export YZMA_LIB=/path/to/libllama.so   # Linux
export YZMA_LIB=/path/to/libllama.dylib  # macOS
set YZMA_LIB=C:\path\to\llama.dll      # Windows
```

### 2. GGUF Models

Download GGUF models from Hugging Face (already integrated via `internal/huggingface/`):

```bash
# Example: SmolLM-135M (tiny model for testing)
wget https://huggingface.co/QuantFactory/SmolLM-135M-GGUF/resolve/main/SmolLM-135M.Q2_K.gguf -P data/models/

# Example: Llama-3.2-1B
wget https://huggingface.co/bartowski/Llama-3.2-1B-Instruct-GGUF/resolve/main/Llama-3.2-1B-Instruct-Q4_K_M.gguf -P data/models/
```

## Implementation Roadmap

### Phase 1: Basic Inference (v3.1.0)
- [ ] Implement `llama.Load()` and `llama.Init()`
- [ ] Model loading with `llama.ModelLoadFromFile()`
- [ ] Context initialization
- [ ] Simple text generation (greedy decoding)
- [ ] Basic tokenization

### Phase 2: Advanced Features (v3.2.0)
- [ ] Streaming generation
- [ ] Sampler chain (temperature, top-p, top-k)
- [ ] Stop sequences
- [ ] Context management (KV cache)
- [ ] Multiple concurrent models

### Phase 3: Optimization (v3.3.0)
- [ ] GPU acceleration (CUDA/Metal)
- [ ] Batch inference
- [ ] Model quantization support (Q2_K, Q4_K_M, Q8_0)
- [ ] Memory optimization
- [ ] Parallel decoding

### Phase 4: Vision Models (v3.4.0)
- [ ] VLM support (multimodal projector)
- [ ] Image encoding
- [ ] Vision-text inference
- [ ] Example: Qwen2.5-VL, LLaVA

## Example Usage (Future)

```go
// Initialize yzma client
yzmaClient, err := yzma.NewClient(yzma.ClientConfig{
    ModelsDir:   "./data/models",
    ContextSize: 2048,
    GPULayers:   35,  // Offload 35 layers to GPU
    Threads:     8,
}, logger)

// Load model
err = yzmaClient.LoadModel(ctx, "Llama-3.2-1B-Instruct-Q4_K_M.gguf")

// Generate text
resp, err := yzmaClient.Generate(ctx, yzma.GenerateRequest{
    ModelPath:   "Llama-3.2-1B-Instruct-Q4_K_M.gguf",
    Prompt:      "What is the capital of France?",
    MaxTokens:   100,
    Temperature: 0.7,
})

fmt.Println(resp.Content)
// Output: "The capital of France is Paris."
```

## Benefits of yzma Over Ollama

1. **No External Service**: Runs directly in process
2. **Lower Latency**: No HTTP overhead
3. **Resource Control**: Fine-grained GPU/CPU allocation
4. **Custom Models**: Full control over model loading
5. **Embedding**: Direct Go integration without REST API
6. **Performance**: Native llama.cpp performance
7. **Portability**: Single binary deployment

## Configuration

```yaml
# configs/dev.yaml (Future)
yzma:
  enabled: true
  models_dir: "./data/models"
  lib_path: "/usr/local/lib/libllama.so"  # Or use YZMA_LIB env var
  context_size: 2048
  gpu_layers: 35  # 0 = CPU only
  threads: 8
  
  # Default sampling
  temperature: 0.7
  top_p: 0.9
  top_k: 40
  repeat_penalty: 1.1
```

## Current Workaround

Until yzma is fully integrated, use:
1. **Ollama** for local inference (existing implementation)
2. **Hugging Face Integration** to download GGUF models
3. **External llama.cpp** with manual model loading

## References

- yzma GitHub: https://github.com/hybridgroup/yzma
- llama.cpp: https://github.com/ggml-org/llama.cpp
- GGUF Format: https://github.com/ggerganov/ggml/blob/master/docs/gguf.md
- Hugging Face GGUF Models: https://huggingface.co/models?library=gguf

## Contributing

Help implement yzma integration:
1. Study llama.cpp API and yzma examples
2. Implement model loading in `internal/yzma/client.go`
3. Add text generation pipeline
4. Create comprehensive tests
5. Add benchmarks vs Ollama

---

**Status**: Placeholder (v3.0.0-dev)  
**Target**: Full implementation in v3.1.0+  
**Priority**: Medium (Ollama works well as interim solution)

