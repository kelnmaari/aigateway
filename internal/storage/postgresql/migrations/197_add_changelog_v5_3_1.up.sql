INSERT INTO changelogs (version, release_date, content) VALUES
('5.3.1', '2026-04-02', '## [5.3.1] - 2026-04-02

### Improved
- Admin tabs overflow: fade-gradient masks on tab navigation edges with hidden scrollbar
- Primary color contrast (WCAG AA): light theme primary darkened from #10A37F (3.4:1) to #0E8F6E (4.6:1)
- Skeleton loading: protected layout and dashboard show skeleton cards instead of text/spinner
- Semantic stat colors: dashboard stat cards use CSS variables with light/dark theme support
- Icon scale standard: documented 6-level icon size standard (xs to 2xl) in app.css')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
