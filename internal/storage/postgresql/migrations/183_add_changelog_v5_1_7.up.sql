INSERT INTO changelogs (version, release_date, content) VALUES
('5.1.7', '2026-03-30', '## [5.1.7] - 2026-03-30

### Added

- **TEI CPU Mode**: Text Embeddings Inference now supports CPU-only execution. New `tei_cpu_mode` checkbox — when enabled, uses `cpu-1.8` image with no GPU passthrough. Useful when GPU is fully occupied by LLM inference and embeddings can run on CPU.

### Fixed

- **TEI Image Selection Bug**: Default image was `89-1.8` (GPU cc 8.9), but CPU path never actually used a CPU image. Fixed: empty GPUDevice or TEICPUMode=true now correctly selects `cpu-1.8`.')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
