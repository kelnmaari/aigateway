INSERT INTO changelogs (version, release_date, content) VALUES
('4.11.0', '2026-02-17', '## [4.11.0] - 2026-02-17

### Added
- **GitLab**: Tool-based review mode — LLM uses structured tool calling instead of raw JSON output for more reliable code reviews
- **GitLab**: ReviewMode setting per project (Standard / Per-file / Tool-based)

### Technical
- Output tools: report_issue, report_suggestion, set_review_summary, finish_review
- ReviewCollector accumulates results from tool calls with validation
- Fallback to JSON parsing when model returns text
- Backward compatible: PerFileReview maps to per_file mode')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
