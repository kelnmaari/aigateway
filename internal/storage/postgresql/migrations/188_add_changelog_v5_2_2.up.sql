INSERT INTO changelogs (version, release_date, content) VALUES
('5.2.2', '2026-03-30', '## [5.2.2] - 2026-03-30

### Fixed
- **ONNX Export optimum 2.x compatibility**: pip install now tries `optimum[onnxruntime]` first (optimum 2.x) and falls back to `optimum[exporters]` (optimum 1.x). Eliminates "does not provide the extra exporters" install failure.

### Technical
- onnx_exporter.go — fallback pip install chain for optimum v1/v2 compatibility')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
