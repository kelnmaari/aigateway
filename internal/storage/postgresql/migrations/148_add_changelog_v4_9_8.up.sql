INSERT INTO changelogs (version, release_date, content) VALUES
('4.9.8', '2026-02-02', '## [4.9.8] - 2026-02-02

### Fixed

- **Secrets Scanner UI**: Fixed field name mismatches in user GitLab integration page
  - `secrets` → `findings` (consistent with backend `SecretsScanResult.findings`)
  - `files_scanned` → `summary.files_affected`
  - Added `chunks_scanned` metric display
  - Labels updated: "Total Secrets" → "Total Findings", "Files Scanned" → "Files Affected"

### Technical

- Full audit of all scanner types confirmed Quality was the only other mismatch (fixed in 4.9.7)
- Dead Code, Docs, Tests, Dependencies scanners already using correct field names')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
