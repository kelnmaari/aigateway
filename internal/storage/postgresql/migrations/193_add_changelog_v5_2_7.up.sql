INSERT INTO changelogs (version, release_date, content) VALUES
('5.2.7', '2026-03-30', '## [5.2.7] - 2026-03-30

### Fixed
- **ONNX Export: missing einops dependency**: Added einops to the pip install step. Required by models with MoE or custom attention architectures (e.g. nomic-embed-text-v2-moe).

### Technical
- onnx_exporter.go — added einops to pip install command')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
