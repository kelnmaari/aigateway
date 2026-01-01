INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('4.2.6', '2025-01-01', '## [4.2.6] - 2025-01-01

### Changed

- **Major Dependencies Update**: Complete NPM and Go dependency refresh
  - Tailwind CSS 3.4 → 4.1 (CSS-first configuration)
  - Vite 6.4 → 7.3
  - bits-ui 1.0.0-next → 2.14.4 (stable)
  - tailwind-variants 0.3 → 3.2
  - tailwind-merge 2.6 → 3.4
  - lucide-svelte 0.469 → 0.562
  - @sveltejs/vite-plugin-svelte 5.0 → 6.2
  - eslint-plugin-svelte 2.46 → 3.13
  - @types/node 22.x → 25.x
  - Go: gin 1.10→1.11, validator 10.26→10.30, fsnotify 1.7→1.9

### Security

- **11 CVEs Fixed**: All NPM vulnerabilities resolved
  - vite: 9 CVEs (CVE-2025-32395, CVE-2025-31125, etc.)
  - @sveltejs/kit: CVE-2025-32388 (XSS via tracked search_params)
  - cookie: GHSA-pxg6-pf52-xh8x (out of bounds characters)

### Technical

- Tailwind 4 migration: @import "tailwindcss" + @theme directive
- Removed tailwind.config.ts (CSS-first config)
- Added @tailwindcss/vite plugin to vite.config.ts
- Added svelte-kit sync to CI/CD before build
- Added cookie override in package.json for transitive fix
- Created docs/VULN.md with update roadmap');

