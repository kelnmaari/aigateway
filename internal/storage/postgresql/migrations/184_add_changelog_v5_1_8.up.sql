INSERT INTO changelogs (version, release_date, content) VALUES
('5.1.8', '2026-03-30', '## [5.1.8] - 2026-03-30

### Fixed

- **Log Viewer Freeze / PC Hang**: Container log modal caused browser main thread to block and freeze when logs were large. Four root causes fixed: (1) colorization called on every Svelte render instead of once per fetch; (2) no cache — identical content fully reprocessed every 2s; (3) ANSI escape codes not stripped, confusing regexes and producing broken HTML; (4) cascading regexes on already-HTML content causing nested spans. Replaced with segment-based colorizer that collects ranges on plain text and builds HTML in one pass. Added 2000-line display cap.')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
