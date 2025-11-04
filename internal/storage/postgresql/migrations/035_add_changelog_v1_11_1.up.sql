
INSERT INTO changelogs (version, release_date, content) VALUES
('1.11.1', '2025-10-25', '## [1.11.1] - 2025-10-25

### Added
- **OIDC-01: Keycloak SSO Integration** 🔐
  - OpenID Connect (OIDC) аутентификация для корпоративного Single Sign-On (SSO)
  - Интеграция с Keycloak и другими OIDC providers (Google, Azure AD, Okta)
  - Authorization Code Flow с PKCE для безопасной аутентификации
  - Автоматическое user provisioning при первом входе через SSO
  - Гибкий claims mapping для разных OIDC providers
  - Role-based access control из OIDC groups/roles
  - Session management для OIDC state с защитой от CSRF
  - HTTP endpoints: /api/auth/oidc/login, /api/auth/oidc/callback, /api/auth/oidc/logout

### Technical
- Новые модули: internal/auth/oidc/provider.go - OIDC provider wrapper, internal/auth/oidc/claims.go - OIDC claims structures, internal/api/handlers/oidc.go - HTTP handlers
- Конфигурация: auth.oidc.enabled, auth.oidc.issuer, auth.oidc.client_id, auth.oidc.client_secret, auth.oidc.scopes, auth.oidc.claims mapping, auth.oidc.auto_create_user, auth.oidc.auto_update_user, auth.oidc.default_role
- База данных (Migration v34): users.auth_provider, users.oidc_subject, users.oidc_issuer, индексы для OIDC, unique constraint (issuer, subject)
- Зависимости: coreos/go-oidc v3, golang.org/x/oauth2, gin-contrib/sessions
- Тестирование: 10 unit tests PASS, 5 SKIP (mock OIDC required)

### Security
- CSRF Protection - random state parameter в OAuth2 flow
- ID Token Verification - проверка подписи и claims через coreos/go-oidc
- Session Security - HttpOnly cookies, SameSite=Lax, secure encryption
- Claims Validation - проверка issuer, audience, expiration')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	