INSERT INTO changelogs (version, release_date, content) VALUES
('5.0.0', '2026-03-25', '## [5.0.0] - 2026-03-25

### Changed
- **Icon System Overhaul**: Replaced all icons across the entire Svelte WebUI from lucide-svelte to Font Awesome 7 (SVG+JS). Tree-shaking ensures only used icons are bundled.
- **Favicon Refresh**: New SVG favicon with multiple sizes, PWA manifest icons, and site.webmanifest for AI Gateway branding
- **Emoji Replaced**: All decorative emoji and text symbols replaced with consistent Font Awesome SVG icons

### Technical
- Removed lucide-svelte, added @fortawesome packages (fontawesome-svg-core, free-solid/regular/brands-svg-icons, svelte-fontawesome)
- 30+ Svelte component files migrated to FontAwesomeIcon pattern
- Updated app.html with proper favicon links (SVG, PNG, ICO, Apple Touch, Web Manifest)')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
