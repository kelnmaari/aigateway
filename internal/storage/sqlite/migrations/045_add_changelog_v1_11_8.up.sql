
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.11.8', '2025-10-26', '## [1.11.8] - 2025-10-26

### Added
- **WebUI Complete Restoration**: Full synchronization of functionality from /web_old/ to /web/
  - Dashboard: Restored gradient background, stats grid, recent conversations, tenants list, quick actions
  - Admin Panel: Complete 11-tab interface (Dashboard, Users, API Keys, Files, Models, MCP Servers, Backups, Audit, RBAC, Logs, System)
  - Navigation: Unified navbar.js component across all pages with prominent Admin link
  - Chat: Full feature parity with model panel, context manager, file attachments
  - API Keys: Personal and organization keys management with full CRUD operations
  - Files: Drag & drop upload, filters, grid view, preview functionality
  - Tenants: Organization management with member roles and permissions
  - Profile: User information editing, password change, account management
  - Usage: Statistics and analytics dashboard
  - About: System information and changelogs display
  - MCP: MCP servers management interface

### Changed
- **UI/UX Improvements**: Applied theme.css consistently across all pages for modern, cohesive design
  - Gradient backgrounds for visual appeal
  - Improved button and tab styling with hover effects
  - Better readability with white headings and text shadows
- **Performance Optimization**: Reduced frequent data request intervals
  - GPU monitor: 5s → 10s update frequency
  - Performance monitor: 5s → 10s update frequency
  - Smart monitor management: auto-stop when tab not active

### Technical
- All 13 HTML pages synchronized and updated
- Consistent navigation component across entire application
- Modern CSS with gradient themes and responsive design
- Optimized JavaScript for better performance
- Fixed console errors and improved error handling
');
    