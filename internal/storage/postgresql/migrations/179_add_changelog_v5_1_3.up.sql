INSERT INTO changelogs (version, release_date, content) VALUES
('5.1.3', '2026-03-30', '## [5.1.3] - 2026-03-30

### Fixed

- **HuggingFace Search Broken**: `Search` icon component (lucide-svelte) was not imported on Models Management page, causing `ReferenceError: Search is not defined` when switching to HuggingFace tab. Replaced with already-imported `FontAwesomeIcon` `faMagnifyingGlass`.')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
