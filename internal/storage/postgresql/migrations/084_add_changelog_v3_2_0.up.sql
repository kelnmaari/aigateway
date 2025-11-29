INSERT INTO changelogs (version, release_date, content) VALUES
('3.2.0', '2025-11-29', '## [3.2.0] - 2025-11-29

### Added

- **Svelte WebUI Migration** (SVELTE-01):
  - Complete frontend rewrite from vanilla HTML/JS to SvelteKit 2.x
  - 22+ pages migrated with full TypeScript support
  - Technology Stack:
    - Svelte 5 with Runes API
    - SvelteKit 2.x with adapter-static (SSG)
    - shadcn-svelte UI components
    - Tailwind CSS 4.x with dark/light theming
    - Paraglide JS for i18n (en/ru)
    - lucide-svelte icons
    - svelte-sonner toast notifications
    
  - Pages: Login, Register, Bootstrap, Dashboard, Chat, Settings, API Keys, Tenants, Profile, Usage, Files, RAG, MCP, Downloads, Monitor, About, Admin (Users, Invitations, API Keys, Models, Settings, Backups, Logs)
    
  - Features:
    - Real-time chat with streaming
    - File upload with drag-and-drop
    - Dark/Light theme with localStorage
    - i18n (en/ru) with JSON files
    - Protected routes with AuthGuard
    - Toast notifications
    - Error page handling

- **Build System Enhancement**:
  - `build.ps1 frontend` command
  - `-WebUI` parameter: legacy, svelte, or both
  - 147 static files (0.55 MB total)

- **Configuration**:
  - `server.webui.version` - Switch between legacy and svelte
  - `server.webui.enabled` - Enable/disable WebUI

### Technical

- `web-svelte/` - Complete SvelteKit project
- `web-migration/docs/` - Migration documentation
- `internal/web/embed.go` - Dual UI support')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;

