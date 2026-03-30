INSERT INTO changelogs (version, release_date, content) VALUES
('5.2.4', '2026-03-30', '## [5.2.4] - 2026-03-30

### Fixed
- **ONNX Export version check**: optimum 2.x removed __version__ as a direct module attribute. Verification now uses importlib.metadata.version(''optimum'') which works with all versions.

### Technical
- onnx_exporter.go — replaced optimum.__version__ with importlib.metadata.version(''optimum'')')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
