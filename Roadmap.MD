# 🗺️ Roadmap: AIGateway Platform

> **Текущая версия:** 3.0.5 ✅  
> **Статус:** v3.0.0-v3.0.5 Phase 1-5 COMPLETED (yzma + Hugging Face + VLM + Migration)  
> **Next:** Bonus Features или Production Deployment  
> **Последнее обновление:** 2025-11-07

---

## 🎯 Vision v3.0: Full Independence & VLM Support

**Major Architecture Shift:**

```
v2.x (Ollama-based):          v3.0 (yzma-based):
Client → Proxy → Ollama       Client → Proxy → yzma → GGUF models
         ↓                             ↓
    Network overhead          Direct memory access
    External dependency       Pure Go inference
    Limited VLM support       Full multimodal support
```

**Ключевые преимущества v3.0:**
- ✅ **Pure Go inference** - нет внешних зависимостей (без CGo!)
- ✅ **GGUF models** - прямая работа с моделями с Hugging Face
- ✅ **VLM support** - Vision Language Models (Qwen2.5-VL, LLaVA, etc.)
- ✅ **Hardware acceleration** - CUDA, Vulkan, Metal автоматически
- ✅ **Hugging Face integration** - загрузка моделей прямо из админки
- ✅ **Fine-grained control** - управление памятью, context window, sampling
- ✅ **Single binary deployment** - 1 сервис вместо 2

---

## 📋 История развития (v1.0 → v2.4.7)

### Completed Versions Summary

| Version | Фокус | Ключевые достижения | Статус |
|---------|-------|---------------------|--------|
| **v1.2.0** | Monitoring & Management | OpenTelemetry, Performance metrics, Scheduled reports | ✅ |
| **v1.3.0** | Multi-Tenancy & UX | User management, Tenants, Multi-tenant isolation | ✅ |
| **v1.4.0** | WebUI & Quality | Chat UI enhancements, Code quality, WebUI v2 | ✅ |
| **v1.5.0** | Operations | Config management, Logging, Maintenance tools | ✅ |
| **v1.6.0** | Observability | OpenTelemetry, Profiling, Reports automation | ✅ |
| **v1.9.0** | Model Management Pro | Dynamic params, Context manager, GPU monitoring, CPU cache optimization | ✅ |
| **v1.10.0** | File Storage & RAG | Universal storage, PDF/DOCX extractors, File attachments | ✅ |
| **v1.11.0** | Enterprise Suite | OIDC SSO (Keycloak), RBAC, LDAP/AD, Audit logging | ✅ |
| **v1.12.0** | Performance & Quotas | Rate limiting, Usage quotas, Optimization | ✅ |
| **v1.13.0** | Advanced RAG | Vector search, Embeddings, Document chunking, Multi-source | ✅ |
| **v2.0.0-v2.4.0** | Production Ready | MCP protocol, Desktop client, Advanced RAG orchestrator | ✅ |
| **v2.4.7** | Codebase Quality | PostgreSQL support (142 migrations), Reversible migrations, Code refactoring | ✅ |
| **v2.6.0** | HTMX Migration | User Management, RBAC UI, Dashboard stats (parallel loading) | ✅ **NEW!** |

**Итого реализовано:**
- 🎯 **13 major versions** (v1.2 → v2.6)
- 📦 **100+ features** implemented
- 🗄️ **PostgreSQL + SQLite** dual storage
- 🔐 **Enterprise-grade security** (OIDC, LDAP, RBAC, Audit)
- 🤖 **Agentic AI** (MCP protocol, tool execution)
- 🎨 **Modern UI** (HTMX + Vanilla JS + Svelte desktop)
- 📊 **Full observability** (OpenTelemetry, metrics, GPU monitoring)

Детальная история см. в [CHANGELOG.md](CHANGELOG.md)

---

## 🚀 v3.0.0 - yzma Integration & Hugging Face (MAJOR RELEASE)

**Приоритет:** CRITICAL  
**Оценка времени:** 22-30 часов  
**Breaking changes:** YES (миграция с Ollama на yzma)  
**Версионирование:** v3.0.1-v3.0.4

### 🎯 Goals

1. **Полная независимость** от Ollama
2. **VLM support** для multimodal AI
3. **Hugging Face integration** для управления моделями
4. **Pure Go inference** с hardware acceleration

### 📊 Architecture Changes

**Новая структура:**

```
internal/
  inference/
    yzma/
      client.go          # yzma library wrapper
      inference.go       # text & VLM inference
      sampler.go         # sampling strategies
      context.go         # context management
      loader.go          # model loading/caching
  huggingface/
    client.go            # HF API client
    downloader.go        # GGUF downloader (with resume)
    search.go            # model search/filter
    metadata.go          # model metadata extraction
  models/
    manager.go           # GGUF model lifecycle
    cache.go             # LRU cache для loaded models
    storage.go           # model files storage

data/
  models/                # GGUF files storage
    llama3-8b-q4.gguf
    qwen2.5-vl-3b.gguf
    mmproj-qwen.gguf     # VLM projectors
```

---

### Phase 1: yzma Core Integration (8-10ч) → v3.0.1 ✅ ЗАВЕРШЕНО

**YZMA-01: yzma Library Integration** (3-4ч) ✅

- [x] Add yzma dependency (`github.com/hybridgroup/yzma`)
- [x] Create `internal/yzma/client.go` (использует llama.cpp через yzma)
- [x] Implement model loading:
  ```go
  type YzmaClient struct {
      libPath    string
      loadedModels map[string]*YzmaModel
  }
  
  func (c *YzmaClient) LoadModel(path string, params ModelParams) error
  func (c *YzmaClient) UnloadModel(modelID string) error
  ```
- [x] Context management (n_ctx, n_batch, n_threads)
- [x] Hardware acceleration detection (CUDA, Vulkan, Metal) - через yzma auto-detect
- [x] Memory management (model offloading to GPU layers) - через llama.cpp

**YZMA-02: Text Inference Implementation** (3-4ч) ✅

- [x] Text generation через yzma (реализовано в `internal/yzma/client.go`)
- [x] Token sampling strategies:
  - [x] Greedy sampling
  - [x] Top-K, Top-P
  - [x] Temperature control
  - [x] Repetition penalty (min_p)
- [x] Streaming support (token-by-token) - через callbacks
- [x] Stop sequences handling
- [x] Token counting для usage tracking

**YZMA-03: OpenAI-Compatible API Adapter** (2-3ч) ✅

- [x] Адаптер `/v1/yzma/chat/completions` (реализовано в `internal/api/handlers/yzma_handler.go`)
- [x] Streaming SSE для chat completions
- [x] OpenAI format conversion (ChatCompletionRequest → yzma prompt)
- [x] Error handling & fallbacks
  
  **Acceptance Criteria:**
- ✅ Загрузка GGUF models из `data/models/`
- ✅ Text generation работает
- ✅ OpenAI API compatibility сохранен
- ✅ Streaming работает корректно
- ✅ GPU offloading активен (если доступно)

**Файлы:**
- `internal/yzma/client.go` - yzma wrapper (420 строк)
- `internal/api/handlers/yzma_handler.go` - OpenAI API adapter (460 строк)
- `internal/config/config.go` - YzmaConfig struct
- `configs/dev.yaml` - yzma configuration section

---

### Phase 2: Hugging Face Integration (6-8ч) → v3.0.2 ✅ ЗАВЕРШЕНО

**HF-01: Hugging Face Client** (3-4ч) ✅

- [x] Implement HF API client (реализовано в `internal/huggingface/client.go`):
  ```go
  type HFClient struct {
      apiToken string
      baseURL  string
  }
  
  func (c *HFClient) SearchModels(filters ModelFilters) ([]*HFModel, error)
  func (c *HFClient) GetModelInfo(modelID string) (*HFModelInfo, error)
  func (c *HFClient) ListFiles(modelID string) ([]*HFFile, error)
  ```
- [x] Model search с фильтрами:
  - [x] Model type (text-generation, image-text-to-text)
  - [x] Quantization (Q4_K_M, Q8_0, FP16)
  - [x] Size/parameters
  - [x] License
  - [x] Language
- [x] Model metadata extraction (config.json, README)
- [x] GGUF file detection (filter only `.gguf` files)

**HF-02: Model Downloader** (3-4ч) ✅

- [x] Streaming download с progress tracking (реализовано в `internal/huggingface/downloader.go`):
  ```go
  func (c *HFClient) DownloadFile(ctx context.Context, fileURL string, opts DownloadOptions) error
  ```
- [x] Resume support (HTTP Range requests)
- [x] Parallel downloads для multi-file models (goroutines)
- [x] SHA256 checksum verification
- [x] Progress reporting через HTMX polling
- [x] Automatic retry на network errors (3 попытки)
- [x] Disk space check перед download
  
  **Acceptance Criteria:**
- ✅ Search Hugging Face models
- ✅ Download GGUF files с progress bar
- ✅ Resume interrupted downloads
- ✅ Verify checksums
- ✅ Graceful error handling

**Файлы:**
- `internal/huggingface/client.go` - HF API client (340 строк)
- `internal/huggingface/downloader.go` - Download manager (480 строк)
- `internal/config/config.go` - HuggingFaceConfig struct
- `configs/dev.yaml` - huggingface configuration section

---

### Phase 3: Model Management UI (4-6ч) → v3.0.3 ✅ ЗАВЕРШЕНО

**HF-UI-01: Hugging Face Browser** (2-3ч) ✅

- [x] Create `web/huggingface.html` (560 строк HTMX UI)
- [x] Search & filter interface:
  - [x] Text search (model name)
  - [x] Filter by type (LLM, VLM, embedding)
  - [x] Filter by quantization
  - [x] Filter by size (<5GB, 5-10GB, >10GB)
  - [x] Sort by downloads, likes, recent
- [x] Model cards display:
  - [x] Model name & description
  - [x] Parameters count
  - [x] License
  - [x] Downloads count
  - [x] Available GGUF files (with sizes)
- [x] One-click download button

**HF-UI-02: Download Manager** (2-3ч) ✅

- [x] Active downloads list
- [x] Progress bars (real-time via HTMX polling)
- [x] Download speed & ETA
- [x] Pause/Resume/Cancel buttons
- [x] Downloaded models registry
- [x] Disk usage statistics
- [x] Model activation toggle (load/unload через yzma UI)

**YZMA-UI-01: yzma Model Management** (2-3ч) ✅ **BONUS!**

- [x] Create `web/yzma.html` (400 строк HTMX UI)
- [x] Available GGUF models list
- [x] Load/Unload models в память
- [x] Real-time statistics (requests, tokens, loaded models)
- [x] Auto-refresh каждые 5 секунд
- [x] Chat provider selector (Ollama / yzma Local)
- [x] Integration с Model Panel в chat.html

**Example UI Flow:**

```
Admin Panel → Models → Browse Hugging Face
                            ↓
                    [Search: "qwen2.5"]
                    Filter: VLM, Q4_K_M, <5GB
                            ↓
                    Model Cards Grid:
                    ┌─────────────────────┐
                    │ Qwen2.5-VL-3B       │
                    │ 3B params | MIT     │
                    │ Files:              │
                    │  • model-Q4.gguf    │
                    │    (2.1 GB)         │
                    │  • mmproj.gguf      │
                    │    (450 MB)         │
                    │ [Download]          │
                    └─────────────────────┘
                            ↓
                    Download Progress:
                    ████████░░ 80% (1.7/2.1 GB)
                    Speed: 12.5 MB/s | ETA: 30s
                            ↓
                    ✅ Download Complete!
                    Model ready to use
```

**Acceptance Criteria:**
- ✅ Browse HF models в WebUI
- ✅ Download GGUF files с UI
- ✅ Real-time progress updates
- ✅ Manage downloaded models
- ✅ Activate/deactivate models

**Файлы:**
- `web/huggingface.html` - HF Model Browser (560 строк)
- `web/yzma.html` - yzma Model Management (400 строк)
- `internal/api/handlers/ui/huggingface_ui_handler.go` - HF HTMX handler (720 строк)
- `internal/api/handlers/ui/yzma_ui_handler.go` - yzma HTMX handler (210 строк)
- `internal/web/templates/hf_*.html` - HF HTMX templates (4 файла)
- `internal/web/templates/yzma_*.html` - yzma HTMX templates (4 файла)
- `web/js/model-panel.js` - обновлен с provider selector
- `web/css/model-panel.css` - стили для provider selector

---

### Phase 4: VLM Support (4-6ч) → v3.0.4 ✅ ЗАВЕРШЕНО

**VLM-01: Vision Language Models** (2-3ч)

- [x] MultiModal projector support (mmproj GGUF files):
  ```go
  type VLMModel struct {
      textModel *YzmaModel
      mmproj    *YzmaModel  // mmproj file
  }
  
  func (m *VLMModel) ProcessImage(imagePath string) (*ImageEmbedding, error)
  func (m *VLMModel) ChatWithImage(image *ImageEmbedding, prompt string) (string, error)
  ```
- [x] Image preprocessing:
  - Decode (JPEG, PNG, WebP)
  - Resize to model requirements
  - Convert to RGB/tensor format
- [x] Image + Text prompt merging
- [x] Multi-turn conversations с images

**VLM-02: VLM API Endpoints** (2-3ч)

- [x] Extend `/v1/chat/completions`:
  ```json
  {
    "model": "qwen2.5-vl-3b",
    "messages": [
      {
        "role": "user",
        "content": [
          {"type": "text", "text": "What's in this image?"},
          {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,..."}}
        ]
      }
    ]
  }
  ```
- [x] Image URL support (base64 и file:// URLs)
- [x] Multi-image support (multiple images in one prompt)
- [x] VLM-specific parameters (image_size, etc.)

**VLM-03: VLM Testing UI** (1-2ч)

- [x] Chat UI image upload
- [x] Image preview в conversation
- [x] VLM validation & routing
- [x] Provider-based VLM support

**Supported VLM Models:**
- ✅ Qwen2.5-VL (3B, 7B)
- ✅ LLaVA (7B, 13B)
- ✅ Gemma 3 Vision
- ✅ MiniCPM-V (2.6, 4.5)

**Acceptance Criteria:**
- ✅ Load VLM models (text + mmproj)
- ✅ Process images correctly
- ✅ Generate responses для image prompts
- ✅ OpenAI-compatible API для images
- ✅ Multi-turn VLM conversations

**Файлы:**
- `internal/yzma/vlm.go` (430 строк) - VLM client wrapper
- `internal/api/handlers/yzma_handler.go` (+165 строк) - VLM API
- `internal/models/openai.go` (+13 строк) - ContentPart, ImageURL
- `web/js/vlm.js` (260 строк) - VLM UI manager
- `web/css/style.css` (+183 строк) - VLM styles

---

### Phase 5: Migration & Cleanup (4-5ч) → v3.0.5 ✅ ЗАВЕРШЕНО

**MIGRATE-01: Configuration Migration** (1-2ч)

- [x] New config structure:
  ```yaml
  inference:
    backend: "yzma"  # was: "ollama"
    yzma:
      lib_path: "${YZMA_LIB}"  # llama.cpp shared lib
      models_dir: "data/models"
      max_loaded_models: 3
      gpu_layers: 99  # -1 = auto
    
    # Remove ollama config:
    # ollama:
    #   url: "http://localhost:11434"
  ```
- [x] Update `internal/config/config.go`
- [x] Migration script для existing configs
- [x] Environment variables mapping

**MIGRATE-02: Database Schema** (1ч)

- [x] New tables:
  ```sql
  CREATE TABLE gguf_models (
      id TEXT PRIMARY KEY,
      name TEXT NOT NULL,
      file_path TEXT NOT NULL,
      file_size INTEGER,
      architecture TEXT,      -- llama, qwen, gemma
      quantization TEXT,       -- Q4_K_M, Q8_0
      parameters_count INTEGER,
      context_size INTEGER,
      is_vlm BOOLEAN DEFAULT FALSE,
      mmproj_path TEXT,        -- for VLMs
      huggingface_id TEXT,     -- original HF model ID
      downloaded_at TIMESTAMP,
      is_active BOOLEAN DEFAULT TRUE
  );
  
  CREATE TABLE model_downloads (
      id TEXT PRIMARY KEY,
      model_id TEXT,
      status TEXT,            -- downloading, completed, failed
      progress INTEGER,       -- 0-100
      total_size INTEGER,
      downloaded_size INTEGER,
      started_at TIMESTAMP,
      completed_at TIMESTAMP
  );
  ```
- [x] Migrations: 078_gguf_models, 079_model_downloads

**MIGRATE-03: Ollama Cleanup** (2-3ч)

- [x] Made Ollama optional in docker-compose.yml
- [x] Update environment variables (AIGATEWAY_INFERENCE_*)
- [x] Update Docker images (Ollama as fallback, not required)
- [x] Update README.md (v3.0 architecture)
- [x] Preserved Ollama code as fallback backend

**MIGRATE-04: Documentation** (1ч)

- [x] Update README.md - v3.0 architecture
- [x] Update VERSION → 3.0.5
- [x] Create changelog v3.0.5
- [x] Update Roadmap.MD - Phase 5 complete

**Acceptance Criteria:**
- ✅ Config migration script работает
- ✅ Database schema обновлен (gguf_models, model_downloads)
- ✅ Ollama made optional (fallback backend)
- ✅ Документация обновлена
- ✅ Single binary deployment (Ollama optional!)

**Файлы:**
- `internal/config/config.go` (+70 строк) - InferenceConfig, migration logic
- `configs/dev.yaml` (+25 строк) - inference section
- `internal/storage/sqlite/migrations/078_*.sql` - gguf_models table
- `internal/storage/sqlite/migrations/079_*.sql` - model_downloads table
- `internal/storage/sqlite/migrations/080_*.sql` - changelog v3.0.5
- `docker-compose.yml` - Ollama optional, yzma config
- `README.md` - v3.0 architecture, VLM highlights

---

## 🎁 Bonus Features (v3.0.x+)

### BONUS-01: Model Quantization (future)

- Convert HF models → GGUF locally
- Quantize FP16 → Q4_K_M, Q8_0
- In-browser progress UI
- Compare quantization quality

### BONUS-02: Model Benchmarking

- Tokens/sec measurement
- Latency profiling (TTFT, TPS)
- GPU utilization metrics
- Comparison dashboard
- Export benchmark reports

### BONUS-03: Smart Model Caching

- LRU cache для loaded models
- Automatic unload когда memory low
- Pre-warming frequently used models
- Model warmup scheduler

### BONUS-04: Model Fine-tuning UI

- Upload training datasets
- Configure LoRA parameters
- Monitor training progress
- Merge adapters → base model
- Export fine-tuned GGUF

---

## 🔮 Future Versions (v3.1+)

### v3.1.0 - Redis Integration & Caching

**Focus:** Distributed caching, sessions, pub/sub

- Redis integration для model cache
- Session storage (JWT → Redis)
- Pub/sub для multi-instance coordination
- Rate limiting через Redis
- Distributed locks

**Оценка:** 8-10 часов

---

### v3.2.0 - Advanced Model Management

**Focus:** Model versioning, A/B testing, canary deployments

- Model versioning (track model updates)
- A/B testing framework
- Canary deployments для new models
- Automatic fallback к stable models
- Performance comparison dashboards

**Оценка:** 10-12 часов

---

### v3.3.0 - Multi-Model Orchestration

**Focus:** Model routing, ensemble, fallback chains

- Intelligent model routing (по complexity)
- Ensemble predictions (multiple models)
- Fallback chains (primary → backup models)
- Cost optimization (route to cheaper models)
- Quality monitoring

**Оценка:** 12-15 часов

---

### v3.4.0 - Advanced VLM Features

**Focus:** Video analysis, OCR, advanced vision

- Video frame extraction & analysis
- OCR integration (text from images)
- Multi-page document processing
- Visual RAG (retrieve by image similarity)
- Image generation integration (Stable Diffusion)

**Оценка:** 15-18 часов

---

## 📊 Version Comparison

| Feature | v2.x (Ollama) | v3.0 (yzma) |
|---------|---------------|-------------|
| **Inference Engine** | Ollama (external) | yzma (embedded) |
| **Dependencies** | Ollama server required | llama.cpp shared lib |
| **CGo Required** | No | No |
| **Model Format** | GGUF (via Ollama) | GGUF (direct) |
| **VLM Support** | Limited | Full (mmproj) |
| **Model Management** | Ollama CLI | Hugging Face UI |
| **Memory Control** | Ollama decides | Fine-grained |
| **GPU Offloading** | Automatic | Configurable |
| **Deployment** | 2 services | 1 binary |
| **Model Loading** | Ollama API | Direct file access |
| **Custom Sampling** | Limited | Full control |
| **Hardware Acceleration** | CUDA only | CUDA, Vulkan, Metal |

---

## 🎯 Migration Checklist (v2.x → v3.0)

**For Developers:**
- [ ] Install yzma dependency
- [ ] Download llama.cpp shared lib для your platform
- [ ] Update config.yaml (ollama → yzma)
- [ ] Run database migration (v073)
- [ ] Download GGUF models via HF UI
- [ ] Test inference endpoints
- [ ] Remove Ollama from docker-compose (if used)

**For Users:**
- [ ] Backup existing config
- [ ] Stop v2.x server
- [ ] Run v3.0 migration script
- [ ] Download models from Hugging Face
- [ ] Start v3.0 server
- [ ] Test API compatibility
- [ ] Update client SDKs (if needed)

**Breaking Changes:**
- ⚠️ Config format changed (ollama → yzma)
- ⚠️ Models need re-download (Ollama models incompatible)
- ⚠️ API endpoints unchanged (OpenAI compatible)
- ⚠️ Database schema extended (auto-migration)

---

## 📝 Implementation Order

**Recommended sequence:**

1. **YZMA-01** → Core integration (text inference baseline)
2. **YZMA-02** → Text generation (verify functionality)
3. **YZMA-03** → API adapter (OpenAI compatibility)
4. **HF-01** → Hugging Face client (model discovery)
5. **HF-02** → Model downloader (get models)
6. **HF-UI-01** → Browse UI (user-facing)
7. **HF-UI-02** → Download manager (UX)
8. **VLM-01** → VLM inference (multimodal)
9. **VLM-02** → VLM API (expose to clients)
10. **MIGRATE-01** → Config migration (smooth transition)
11. **CLEANUP-01** → Remove Ollama (finalize)

**Parallel work possible:**
- HF-01 + HF-02 можно делать параллельно с YZMA-02
- HF-UI-01 + HF-UI-02 после того как HF-02 готов
- VLM-01 можно начать после YZMA-02

---

## 🔧 Technical Details

### yzma Library

**Source:** [github.com/hybridgroup/yzma](https://github.com/hybridgroup/yzma)

**Key Features:**
- Pure Go (purego + ffi)
- No CGo required
- Latest llama.cpp support
- Hardware acceleration (CUDA, Vulkan, Metal)
- 87%+ llama.cpp API coverage
- Cross-platform (Linux, macOS, Windows)

**Example:**
```go
import "github.com/hybridgroup/yzma/pkg/llama"

llama.Load(libPath)
llama.Init()

model := llama.ModelLoadFromFile(modelFile, llama.ModelDefaultParams())
lctx := llama.InitFromModel(model, llama.ContextDefaultParams())

// Inference...
```

### Supported Hardware

- ✅ **CPU:** x86_64, ARM64
- ✅ **GPU NVIDIA:** CUDA 11.x, 12.x
- ✅ **GPU AMD:** Vulkan
- ✅ **GPU Apple:** Metal (M1, M2, M3)
- ✅ **GPU Intel:** oneAPI (experimental)

### Model Formats

- ✅ **GGUF** (primary format)
- ✅ **Quantizations:** Q2_K, Q3_K_S, Q3_K_M, Q4_K_S, Q4_K_M, Q5_K_S, Q5_K_M, Q6_K, Q8_0, FP16
- ✅ **Architectures:** LLaMA, Qwen, Gemma, Mistral, Phi, and more

---

**Последнее обновление:** 2025-11-07  
**Версия Roadmap:** 3.0  
**Статус:** 🚀 v3.0.0 Planning Complete → Ready for Implementation!
