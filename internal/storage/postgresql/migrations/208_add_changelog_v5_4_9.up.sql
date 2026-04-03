INSERT INTO changelogs (version, release_date, content) VALUES
('5.4.9', '2026-04-02', '## [5.4.9] - 2026-04-02

### Added
- vLLM Allow Long Context checkbox: sets VLLM_ALLOW_LONG_MAX_MODEL_LEN=1 env var to allow max_model_len exceeding max_position_embeddings. Available in Load Model form and Edit Saved Model modal.')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
