INSERT INTO changelogs (version, release_date, content) VALUES
('4.11.1', '2026-02-17', '## [4.11.1] - 2026-02-17

### Fixed
- **Admin UI**: Providers page gracefully handles disabled Model Registry — shows informational banner instead of 404 console error
- **HuggingFace**: Returns HTTP 401 instead of 500 when API token is expired or unauthorized')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
