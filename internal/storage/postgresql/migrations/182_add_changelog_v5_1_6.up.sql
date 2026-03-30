INSERT INTO changelogs (version, release_date, content) VALUES
('5.1.6', '2026-03-30', '## [5.1.6] - 2026-03-30

### Fixed

- **Extra Args JSON Parsing**: `strings.Fields()` split CLI arguments by whitespace only, breaking JSON values passed to `--hf-overrides` and similar flags. Replaced with a shell-like `splitArgs()` parser that respects single and double quoted strings. Now you can pass `--hf-overrides ''{"key":"value"}''` in the Extra Args field of any inference provider (vLLM, SGLang, TGI, TEI, llama.cpp).')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
