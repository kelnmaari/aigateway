INSERT INTO changelogs (version, release_date, content) VALUES
('4.9.7', '2026-02-02', '## [4.9.7] - 2026-02-02

### Fixed

- **Jobs API**: Fixed `jobs: null` response when no jobs exist - now returns empty array `[]`
- **Quality Analysis UI**: Fixed field name mismatches between user and admin UI
  - `score` → `overall_score`
  - `files_count` → `summary.total_files`
  - `issues[]` → `file_scores[].issues` and `recommendations[]`

### Changed

- **Quality Results Visualization**: Completely redesigned quality analysis results display
  - Shows recommendations with priority badges (high/medium/low)
  - Shows issues grouped by file with score per file
  - Shows score breakdown by category (complexity, documentation, security, etc.)
  - Dynamic color coding based on score thresholds')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
