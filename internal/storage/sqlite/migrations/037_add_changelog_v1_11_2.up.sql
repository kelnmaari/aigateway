
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.11.2', '2025-10-25', '## [1.11.2] - 2025-10-25

### Added
- **OIDC-02: Auto-tenant Provisioning from OIDC Groups** 🏢
  - Автоматическое создание tenants из OIDC groups claims (Keycloak, Google, Azure AD)
  - Group → Tenant mapping с двумя режимами: Direct (1:1) и Prefix (path extraction)
  - Auto-provisioning: создание tenants и добавление пользователей при первом логине
  - Role assignment: admin/member роли из OIDC groups
  - Orphaned memberships cleanup (опционально)
  - Tenant name normalization

### Technical
- Модули: internal/auth/oidc/tenants.go (parsing), internal/auth/oidc/provisioner.go (provisioner)
- Configuration: auth.oidc.tenant_provisioning (enabled, auto_create_tenants, sync_on_login, group_mapping)
- Database (Migration v36): UNIQUE INDEX на tenants.name, GetTenantByName method
- Integration: OIDC callback с tenant provisioning, JWT с tenant IDs
- Tests: 16 unit tests (ParseGroups, mapping, normalization)');
	