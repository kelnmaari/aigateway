# OIDC-01: Keycloak SSO Integration

**Версия:** 1.9.0  
**Приоритет:** High  
**Сложность:** High  
**Оценка:** 16-24 часа

## Описание

Интеграция OpenID Connect (OIDC) для Single Sign-On с Keycloak. Пользователи смогут авторизовываться через корпоративный Keycloak сервер вместо создания отдельных учетных записей.

## Проблема

В текущей версии:
- Только локальная аутентификация (username/password)
- Нет интеграции с корпоративным SSO
- Дублирование учетных записей
- Невозможность централизованного управления пользователями
- Отсутствие федеративной аутентификации

## Решение

OIDC/OAuth2 flow для аутентификации через Keycloak.

### Архитектура

```
Browser → [Login] → Keycloak Authorization Server
              ↓
        [OAuth2 Code Grant Flow]
              ↓
     Proxy Server ← [ID Token + Access Token]
              ↓
       [Validate Token] → [Create/Update User] → [Issue JWT]
              ↓
         WebUI (authenticated)
```

## Технические детали

### OAuth2/OIDC Flow

**1. Authorization Code Grant Flow**

```
1. User clicks "Login with Keycloak"
   ↓
2. Redirect to Keycloak:
   GET https://keycloak.example.com/realms/{realm}/protocol/openid-connect/auth
       ?client_id=ollama-proxy
       &redirect_uri=https://proxy.example.com/auth/callback
       &response_type=code
       &scope=openid profile email groups
   ↓
3. User logs in on Keycloak
   ↓
4. Keycloak redirects back:
   GET https://proxy.example.com/auth/callback?code=ABC123
   ↓
5. Exchange code for tokens:
   POST https://keycloak.example.com/realms/{realm}/protocol/openid-connect/token
        code=ABC123
        client_id=ollama-proxy
        client_secret=secret
        grant_type=authorization_code
   ↓
6. Keycloak returns tokens:
   {
     "access_token": "...",
     "id_token": "...",
     "refresh_token": "..."
   }
   ↓
7. Validate ID token → Extract claims → Create/Update user → Issue JWT
```

### Backend Implementation

**1. OIDC Configuration**

```yaml
# config.yaml
auth:
  oidc:
    enabled: true
    provider: "keycloak"
    issuer: "https://keycloak.example.com/realms/myrealm"
    client_id: "ollama-proxy"
    client_secret: "${OIDC_CLIENT_SECRET}"
    redirect_uri: "https://proxy.example.com/auth/callback"
    scopes:
      - openid
      - profile
      - email
      - groups
    
    # Claims mapping
    claims:
      user_id: "sub"
      username: "preferred_username"
      email: "email"
      name: "name"
      groups: "groups"
    
    # Auto-provisioning
    auto_create_user: true
    auto_update_user: true
```

**2. OIDC Provider**

```go
// internal/auth/oidc/provider.go

import (
    "github.com/coreos/go-oidc/v3/oidc"
    "golang.org/x/oauth2"
)

type OIDCProvider struct {
    config       *config.OIDCConfig
    provider     *oidc.Provider
    verifier     *oidc.IDTokenVerifier
    oauth2Config *oauth2.Config
}

func NewOIDCProvider(cfg *config.OIDCConfig) (*OIDCProvider, error) {
    ctx := context.Background()
    
    // Discover OIDC endpoints
    provider, err := oidc.NewProvider(ctx, cfg.Issuer)
    if err != nil {
        return nil, fmt.Errorf("failed to discover OIDC endpoints: %w", err)
    }
    
    // Configure ID token verifier
    verifier := provider.Verifier(&oidc.Config{
        ClientID: cfg.ClientID,
    })
    
    // Configure OAuth2
    oauth2Config := &oauth2.Config{
        ClientID:     cfg.ClientID,
        ClientSecret: cfg.ClientSecret,
        RedirectURL:  cfg.RedirectURI,
        Endpoint:     provider.Endpoint(),
        Scopes:       cfg.Scopes,
    }
    
    return &OIDCProvider{
        config:       cfg,
        provider:     provider,
        verifier:     verifier,
        oauth2Config: oauth2Config,
    }, nil
}

func (p *OIDCProvider) GetAuthURL(state string) string {
    return p.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (p *OIDCProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
    return p.oauth2Config.Exchange(ctx, code)
}

func (p *OIDCProvider) VerifyIDToken(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
    return p.verifier.Verify(ctx, rawIDToken)
}
```

**3. OIDC Handlers**

```go
// internal/api/handlers/oidc.go

type OIDCHandler struct {
    provider *oidc.OIDCProvider
    db       storage.Database
    jwtSvc   *auth.JWTService
    logger   *logrus.Logger
}

// Initiate OIDC login
func (h *OIDCHandler) Login(c *gin.Context) {
    // Generate state parameter (CSRF protection)
    state := generateRandomState()
    
    // Store state in session
    session := sessions.Default(c)
    session.Set("oidc_state", state)
    session.Save()
    
    // Redirect to Keycloak
    authURL := h.provider.GetAuthURL(state)
    c.Redirect(http.StatusFound, authURL)
}

// Handle OIDC callback
func (h *OIDCHandler) Callback(c *gin.Context) {
    // Validate state (CSRF)
    session := sessions.Default(c)
    savedState := session.Get("oidc_state")
    receivedState := c.Query("state")
    
    if savedState != receivedState {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
        return
    }
    
    // Exchange code for tokens
    code := c.Query("code")
    token, err := h.provider.ExchangeCode(c.Request.Context(), code)
    if err != nil {
        h.logger.WithError(err).Error("Failed to exchange code")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "token exchange failed"})
        return
    }
    
    // Extract ID token
    rawIDToken, ok := token.Extra("id_token").(string)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "no id_token"})
        return
    }
    
    // Verify ID token
    idToken, err := h.provider.VerifyIDToken(c.Request.Context(), rawIDToken)
    if err != nil {
        h.logger.WithError(err).Error("Failed to verify ID token")
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid id_token"})
        return
    }
    
    // Extract claims
    var claims OIDCClaims
    if err := idToken.Claims(&claims); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse claims"})
        return
    }
    
    // Create or update user
    user, err := h.provisionUser(c.Request.Context(), &claims)
    if err != nil {
        h.logger.WithError(err).Error("Failed to provision user")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "user provisioning failed"})
        return
    }
    
    // Issue JWT token
    jwtToken, err := h.jwtSvc.GenerateToken(user.ID, user.Role)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
        return
    }
    
    // Redirect to WebUI with token
    c.Redirect(http.StatusFound, fmt.Sprintf("/dashboard?token=%s", jwtToken))
}

type OIDCClaims struct {
    Subject          string   `json:"sub"`
    PreferredUsername string  `json:"preferred_username"`
    Email            string   `json:"email"`
    EmailVerified    bool     `json:"email_verified"`
    Name             string   `json:"name"`
    GivenName        string   `json:"given_name"`
    FamilyName       string   `json:"family_name"`
    Groups           []string `json:"groups"`
}

func (h *OIDCHandler) provisionUser(ctx context.Context, claims *OIDCClaims) (*models.User, error) {
    // Check if user exists (by OIDC subject)
    user, err := h.db.GetUserByOIDCSubject(ctx, claims.Subject)
    if err == nil {
        // User exists, update
        if h.provider.config.AutoUpdateUser {
            user.Username = claims.PreferredUsername
            user.Email = claims.Email
            user.FullName = claims.Name
            if err := h.db.UpdateUser(ctx, user); err != nil {
                return nil, err
            }
        }
        return user, nil
    }
    
    // User doesn't exist, create
    if !h.provider.config.AutoCreateUser {
        return nil, errors.New("user auto-creation disabled")
    }
    
    user = &models.User{
        ID:           uuid.New().String(),
        Username:     claims.PreferredUsername,
        Email:        claims.Email,
        FullName:     claims.Name,
        Role:         models.UserRoleUser, // Default role
        AuthProvider: models.AuthProviderOIDC,
        OIDCSubject:  &claims.Subject,
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
    }
    
    if err := h.db.CreateUser(ctx, user); err != nil {
        return nil, err
    }
    
    h.logger.WithFields(logrus.Fields{
        "user_id":  user.ID,
        "username": user.Username,
        "oidc_sub": claims.Subject,
    }).Info("User auto-provisioned via OIDC")
    
    return user, nil
}
```

**4. Database Schema Updates**

```sql
-- Add OIDC fields to users table
ALTER TABLE users ADD COLUMN auth_provider TEXT DEFAULT 'local'; -- 'local', 'oidc', 'ldap'
ALTER TABLE users ADD COLUMN oidc_subject TEXT UNIQUE; -- OIDC 'sub' claim
ALTER TABLE users ADD COLUMN oidc_issuer TEXT; -- OIDC issuer URL

CREATE INDEX idx_users_oidc_subject ON users(oidc_subject) WHERE oidc_subject IS NOT NULL;
CREATE INDEX idx_users_auth_provider ON users(auth_provider);
```

### Frontend Integration

**Login Page:**

```html
<!-- web/login.html -->
<div class="login-container">
  <h2>Login</h2>
  
  <!-- OIDC Login Button -->
  <button class="btn-oidc" onclick="loginWithOIDC()">
    <img src="/images/keycloak-icon.svg" />
    Login with Keycloak
  </button>
  
  <div class="divider">OR</div>
  
  <!-- Local Login Form -->
  <form id="local-login-form">
    <input type="text" name="username" placeholder="Username" />
    <input type="password" name="password" placeholder="Password" />
    <button type="submit">Login with Password</button>
  </form>
</div>

<script>
function loginWithOIDC() {
  window.location.href = '/auth/oidc/login';
}
</script>
```

### Admin UI Configuration

**Admin Panel → Settings → Authentication:**

```html
<!-- web/admin.html → Authentication tab -->
<div class="auth-settings">
  <h3>OIDC / Keycloak Configuration</h3>
  
  <label>
    <input type="checkbox" id="oidc-enabled" />
    Enable OIDC Authentication
  </label>
  
  <input type="text" id="oidc-issuer" placeholder="Issuer URL" />
  <input type="text" id="oidc-client-id" placeholder="Client ID" />
  <input type="password" id="oidc-client-secret" placeholder="Client Secret" />
  <input type="text" id="oidc-redirect-uri" placeholder="Redirect URI" />
  
  <label>
    <input type="checkbox" id="oidc-auto-create" />
    Auto-create users on first login
  </label>
  
  <label>
    <input type="checkbox" id="oidc-auto-update" />
    Auto-update user info on login
  </label>
  
  <button onclick="saveOIDCConfig()">Save Configuration</button>
  <button onclick="testOIDCConnection()">Test Connection</button>
</div>
```

## Security

**1. State Parameter (CSRF Protection)**
- Random state generated для каждого login request
- Validated при callback

**2. ID Token Validation**
- Signature verification
- Issuer validation
- Audience validation (client_id)
- Expiration check

**3. PKCE (optional, для public clients)**
- Code challenge/verifier
- Enhanced security для SPA

## Требования

### Функциональные

1. ✅ OIDC Authorization Code Flow
2. ✅ Keycloak integration
3. ✅ JWT token issuance после OIDC login
4. ✅ User auto-provisioning
5. ✅ User auto-update (email, name)
6. ✅ Group claims extraction (для OIDC-02)
7. ✅ Admin UI для OIDC configuration
8. ✅ "Login with Keycloak" button
9. ✅ Fallback to local auth (dual mode)
10. ✅ Test connection функция в Admin UI

### Нефункциональные

1. **Security**
   - State parameter для CSRF protection
   - ID token signature verification
   - Secure client secret storage

2. **Performance**
   - Login flow < 3s
   - Token validation < 100ms

3. **Reliability**
   - Graceful fallback если Keycloak unavailable
   - Clear error messages для users

## Acceptance Criteria

- [ ] Пользователь может логиниться через Keycloak
- [ ] OIDC flow (redirect → login → callback) работает корректно
- [ ] ID token валидируется и claims извлекаются
- [ ] User auto-provisioning создает нового user
- [ ] Existing users обновляются (email, name)
- [ ] JWT токен выдается после успешного OIDC login
- [ ] Admin может настроить OIDC через WebUI
- [ ] Test connection проверяет OIDC endpoints
- [ ] Fallback на local auth работает если OIDC disabled
- [ ] Security: state validation, token verification
- [ ] Unit tests для OIDC provider, handlers
- [ ] Integration tests для full OIDC flow

## Риски и зависимости

### Риски

1. **Keycloak unavailable** - SSO down
   - Mitigation: Fallback to local auth, clear error message

2. **Token expiration** - long sessions
   - Mitigation: Refresh token support (future)

3. **Claims mapping** - non-standard claims
   - Mitigation: Configurable claims mapping

### Зависимости

1. **coreos/go-oidc** - OIDC library (github.com/coreos/go-oidc/v3/oidc)
2. **golang.org/x/oauth2** - OAuth2 library
3. **Keycloak server** - external dependency

## Связанные задачи

- **OIDC-02**: Auto-tenant from Groups - использует group claims
- **LDAP-01**: LDAP Integration - альтернативный auth provider
- **AUDIT-01**: Enhanced Audit Logging - логирование OIDC events

## Примечания

- Только OIDC provider (Keycloak) в v1.9.0
- Другие providers (Azure AD, Google, Okta) - в future versions
- Refresh token support - в future versions
- MFA через Keycloak (не в proxy)

---

**Статус:** 📋 Planned for v1.9.0  
**Последнее обновление:** 2025-10-11


