INSERT INTO changelogs (version, release_date, content) VALUES
('5.0.2', '2026-03-26', '## [5.0.2] - 2026-03-26

### Changed
- **Go 1.26.1**: Upgraded to Go 1.26.1, ran `go fix ./...` across entire codebase
- **Modern Go Idioms**: Replaced manual min/max patterns with builtins, for-range-int loops, and `new(T(v))` composite literals
- **Removed Dead Code**: Removed unused `Float32Ptr()` and `BoolPtr()` helpers — all call sites inlined by `go fix`
- **WebUI Cleanup**: Removed unused `faClipboard` import

### Technical
- 6 Go files modernized by `go fix`')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
