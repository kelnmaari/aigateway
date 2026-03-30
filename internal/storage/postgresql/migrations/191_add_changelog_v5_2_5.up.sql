INSERT INTO changelogs (version, release_date, content) VALUES
('5.2.5', '2026-03-30', '## [5.2.5] - 2026-03-30

### Fixed
- **ONNX Export dtype options**: Removed int8/uint8 (unsupported in optimum 2.x). Now fp32/fp16/bf16 only. Default changed to fp32.
- **ONNX Export task/dtype descriptions**: Both dropdowns now include human-readable explanations.')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
