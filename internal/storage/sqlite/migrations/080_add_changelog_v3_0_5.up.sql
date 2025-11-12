INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('3.0.5', '2025-11-07', '## [3.0.5] - 2025-11-07

### Added

- **🔧 Configuration Migration** (Phase 5: MIGRATE-01):
  - `InferenceConfig` struct - unified backend selector
  - `inference.backend` configuration: "yzma" (default), "ollama" (fallback), "auto"
  - `migrateToInferenceConfig()` - automatic migration from old config format
  - Backward compatibility preserved for old `ollama` and `yzma` top-level configs
  - New defaults: `inference.max_loaded_models=3`, `inference.gpu_layers=-1` (auto)

- **💾 Database Schema for GGUF Models** (Phase 5: MIGRATE-02):
  - `gguf_models` table - tracking downloaded GGUF models
    - Fields: id, name, file_path, file_size, architecture, quantization
    - VLM support: is_vlm, mmproj_path
    - Hugging Face metadata: huggingface_id, huggingface_file
    - Usage tracking: last_loaded_at, load_count
  - `model_downloads` table - tracking download progress
    - Status: pending, downloading, paused, completed, failed, cancelled
    - Progress tracking: downloaded_size, total_size, download_speed, eta_seconds
    - SHA256 verification: sha256_hash, sha256_verified
    - Resume support: temp_file_path, paused_at, resumed_at, retry_count

### Changed

- **🐳 Docker Deployment** (Phase 5: MIGRATE-03):
  - Ollama service now OPTIONAL (commented out by default)
  - Added `AIGATEWAY_INFERENCE_BACKEND=yzma` environment variable
  - Added `YZMA_LIB` environment variable for llama.cpp library path
  - Added `/app/data/models` volume mount for GGUF models
  - Removed `depends_on: ollama` requirement
  - Single binary deployment - no external dependencies!

- **📖 Documentation Updates** (Phase 5: MIGRATE-04):
  - README.md - Updated to v3.0 architecture
  - Emphasis on local-first inference with yzma
  - VLM support highlighted
  - Ollama marked as optional fallback backend
  - Updated version badge to 3.0.5

### Technical

- **Config Migration**: Automatic detection of old config format
- **Backward Compatibility**: Old `ollama.*` and `yzma.*` configs still work
- **Environment Variables**: New `AIGATEWAY_INFERENCE_*` prefix
- **Database Migrations**: 3 new reversible migrations (078-080)
- **Docker Compose**: Ollama can be enabled with `docker-compose --profile ollama up`

### Migration Guide

**From v3.0.4 to v3.0.5:**

1. **Config Migration** (automatic):
   - Old format detected automatically
   - Console output shows migration progress
   - No action required

2. **Docker Users**:
   ```bash
   # v3.0.5 default (yzma only):
   docker-compose up
   
   # With Ollama fallback (optional):
   docker-compose --profile ollama up
   ```

3. **Environment Variables** (optional update):
   ```bash
   # New format (recommended):
   AIGATEWAY_INFERENCE_BACKEND=yzma
   AIGATEWAY_INFERENCE_MAX_LOADED_MODELS=3
   YZMA_LIB=/path/to/libllama.so
   
   # Old format (still supported):
   AIGATEWAY_OLLAMA_URL=http://localhost:11434
   ```

### Deprecation Notices

- ⚠️ **Top-level `ollama` config**: Use `inference.ollama` instead (will be removed in v3.1.0)
- ⚠️ **Top-level `yzma` config**: Use `inference.yzma` instead (will be removed in v3.1.0)
- ⚠️ **Environment variable `AIGATEWAY_OLLAMA_*`**: Use `AIGATEWAY_INFERENCE_OLLAMA_*` (will be removed in v3.1.0)');

