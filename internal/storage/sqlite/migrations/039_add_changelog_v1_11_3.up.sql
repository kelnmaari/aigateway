
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.11.3', '2025-10-25', '## [1.11.3] - 2025-10-25

### Added
- **LDAP-01: LDAP/Active Directory Integration** 🔐
  - LDAP Bind Authentication для корпоративных LDAP/AD серверов
  - User/Group Search с настраиваемыми фильтрами (OpenLDAP, Active Directory)
  - Auto-provisioning users при первом логине
  - Auto-update users синхронизация email/full name
  - Tenant provisioning из LDAP groups (reuse OIDC-02 logic)
  - TLS/LDAPS support с StartTLS и certificate validation
  - Admin detection на основе LDAP groups
  - Test connection endpoint для admin (/api/auth/ldap/test)

### Technical
- Модули: internal/auth/ldap/client.go (LDAP client), internal/api/handlers/ldap.go (handler)
- Configuration: auth.ldap (URL, bind credentials, user/group search, TLS, provisioning)
- Database (Migration v38): ALTER TABLE users ADD COLUMN ldap_dn TEXT UNIQUE
- API Routes: POST /api/auth/ldap/login (public), GET /api/auth/ldap/test (admin)
- Integration: JWT с tenant IDs, tenant provisioner из OIDC-02
- Tests: 17 unit tests (config, authentication, isAdminGroup)
- Support: OpenLDAP, Active Directory, FreeIPA');
	