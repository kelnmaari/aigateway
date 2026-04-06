INSERT INTO changelogs (version, release_date, content) VALUES
('5.5.0', '2026-04-02', '## [5.5.0] - 2026-04-02

### Added
- TurboQuant KV-cache compression: two Docker images
  - vllm-turboquant (Alberto-Codes plugin) — basic KV cache compression up to 3.76x via --attention-backend CUSTOM
  - vllm-turboquant-plus (Varjosoft Oy fork) — TQ3 weight compression + KV cache + MoE expert pruning (REAP)
- New ModelSpec fields: VLLMTurboQuantEnabled / KBits / VBits, plus tq3/tq_k4v3/tq_k4v2 KV Cache Dtype options
- Backend automatically translates tq* KV cache dtypes to TurboQuant+ environment variables (TQ_WEIGHT_BITS, TQ_KV_ENABLED, TQ_K_BITS, TQ_V_BITS, TQ_NORM_CORRECTION) without passing --kv-cache-dtype CLI flag to vLLM
- Disable Reasoning checkbox: --disable-reasoning to keep <think> tags in content field (for clients like OpenCode, Kilo Code)
- Server config inference.merge_reasoning_content: SSE proxy merges reasoning_content into content with <think> tags
- Allow Long Context checkbox: VLLM_ALLOW_LONG_MAX_MODEL_LEN=1 env var

### Fixed
- TurboQuant + fp8 KV cache conflict — backend HTTP 400 + UI button disabled with tooltip
- UI warnings for tq* dtype without turboquant-plus image and for redundant basic+plus combinations

### Technical
- New entrypoint wrapper for turboquant-plus image applies Python-API patches (enable_weight_quantization, patch_vllm_attention, enable_reap_pruning) before vllm serve
- Helper validateTurboQuantConfig as single source of truth for conflict detection
- README rewritten for v5 with TurboQuant section and Docker build commands')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
