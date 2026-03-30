INSERT INTO changelogs (version, release_date, content) VALUES
('5.2.6', '2026-03-30', '## [5.2.6] - 2026-03-30

### Fixed
- **ONNX Export: custom architecture models**: Added --trust-remote-code flag to optimum-cli export onnx. Required for models with non-standard architectures (e.g. nomic-embed-text-v2-moe uses nomic-bert-2048).

### Technical
- onnx_exporter.go — added --trust-remote-code to optimum-cli export command')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
