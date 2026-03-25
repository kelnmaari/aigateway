INSERT INTO changelogs (version, release_date, content) VALUES
('5.0.1', '2026-03-25', '## [5.0.1] - 2026-03-25

### Fixed
- **Missed Icon Replacements**: Fixed remaining lucide-svelte component references in admin/models, admin/downloads, admin/gitlab/[id], and user gitlab/[id] pages
- **Dynamic Icon Functions**: Fixed getStatusIcon() and getSeverityIcon() returning old lucide components instead of FA icon definitions

### Technical
- Replaced all remaining old icon components with FontAwesomeIcon pattern
- Updated dynamic icon functions to return FA icon definitions')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
