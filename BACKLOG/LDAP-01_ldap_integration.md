# LDAP-01: LDAP/Active Directory Integration

**Версия:** 1.9.0  
**Приоритет:** Medium  
**Сложность:** Medium  
**Оценка:** 12-16 часов

## Описание

Интеграция LDAP/Active Directory для аутентификации и синхронизации пользователей. Пользователи смогут логиниться с корпоративными LDAP credentials, автоматическая синхронизация групп в tenants.

## Проблема

После OIDC-01:
- Не все организации используют OIDC/OAuth2
- Legacy системы работают с LDAP/AD
- Нужна альтернатива для LDAP-only окружений
- Нет интеграции с Active Directory

## Решение

LDAP bind authentication + optional user/group sync.

### Архитектура

```
User (username/password) → Proxy Server
                               ↓
                      [LDAP Bind Authentication]
                               ↓
                      LDAP/AD Server
                               ↓
                      [Auth Success/Fail]
                               ↓
                      [Optional: Fetch Groups]
                               ↓
                      [Create/Update User]
                               ↓
                      [Issue JWT Token]
```

## Технические детали

### Configuration

```yaml
# config.yaml
auth:
  ldap:
    enabled: true
    
    # LDAP Server
    url: "ldap://ldap.example.com:389" # or ldaps://ldap.example.com:636
    bind_dn: "cn=admin,dc=example,dc=com"
    bind_password: "${LDAP_BIND_PASSWORD}"
    
    # User search
    user_base_dn: "ou=users,dc=example,dc=com"
    user_filter: "(uid={username})" # or "(sAMAccountName={username})" for AD
    user_id_attribute: "uid" # or "sAMAccountName"
    user_email_attribute: "mail"
    user_name_attribute: "cn"
    
    # Group search (optional)
    group_base_dn: "ou=groups,dc=example,dc=com"
    group_filter: "(member={userdn})" # or "(memberOf={userdn})"
    group_name_attribute: "cn"
    
    # TLS/SSL
    start_tls: false
    skip_verify: false # Don't skip in production
    ca_cert_file: "/path/to/ca.crt"
    
    # Auto-provisioning
    auto_create_user: true
    auto_update_user: true
    
    # Group → Tenant mapping (similar to OIDC-02)
    group_to_tenant:
      enabled: true
      auto_create_tenants: true
      admin_groups:
        - "Domain Admins"
        - "Engineering Admins"
```

### Implementation

**1. LDAP Client**

```go
// internal/auth/ldap/client.go

import "github.com/go-ldap/ldap/v3"

type LDAPClient struct {
    config *config.LDAPConfig
    logger *logrus.Logger
}

func NewLDAPClient(cfg *config.LDAPConfig) (*LDAPClient, error) {
    return &LDAPClient{
        config: cfg,
    }, nil
}

func (c *LDAPClient) Authenticate(username, password string) (*LDAPUser, error) {
    // Connect to LDAP
    conn, err := c.connect()
    if err != nil {
        return nil, fmt.Errorf("failed to connect: %w", err)
    }
    defer conn.Close()
    
    // Bind with service account
    if err := conn.Bind(c.config.BindDN, c.config.BindPassword); err != nil {
        return nil, fmt.Errorf("failed to bind: %w", err)
    }
    
    // Search for user
    searchRequest := ldap.NewSearchRequest(
        c.config.UserBaseDN,
        ldap.ScopeWholeSubtree,
        ldap.NeverDerefAliases,
        0, 0, false,
        strings.Replace(c.config.UserFilter, "{username}", ldap.EscapeFilter(username), -1),
        []string{"dn", c.config.UserIDAttribute, c.config.UserEmailAttribute, c.config.UserNameAttribute},
        nil,
    )
    
    searchResult, err := conn.Search(searchRequest)
    if err != nil {
        return nil, fmt.Errorf("failed to search user: %w", err)
    }
    
    if len(searchResult.Entries) == 0 {
        return nil, errors.New("user not found")
    }
    
    if len(searchResult.Entries) > 1 {
        return nil, errors.New("multiple users found")
    }
    
    entry := searchResult.Entries[0]
    userDN := entry.DN
    
    // Try to bind as user (authenticate)
    if err := conn.Bind(userDN, password); err != nil {
        return nil, fmt.Errorf("authentication failed: %w", err)
    }
    
    // Rebind as service account for group search
    if err := conn.Bind(c.config.BindDN, c.config.BindPassword); err != nil {
        return nil, fmt.Errorf("failed to rebind: %w", err)
    }
    
    // Build user object
    user := &LDAPUser{
        DN:       userDN,
        Username: entry.GetAttributeValue(c.config.UserIDAttribute),
        Email:    entry.GetAttributeValue(c.config.UserEmailAttribute),
        FullName: entry.GetAttributeValue(c.config.UserNameAttribute),
    }
    
    // Fetch groups (optional)
    if c.config.GroupBaseDN != "" {
        groups, err := c.getUserGroups(conn, userDN)
        if err != nil {
            c.logger.WithError(err).Warn("Failed to fetch user groups")
        } else {
            user.Groups = groups
        }
    }
    
    return user, nil
}

func (c *LDAPClient) connect() (*ldap.Conn, error) {
    // Parse URL
    u, err := url.Parse(c.config.URL)
    if err != nil {
        return nil, err
    }
    
    // Connect
    var conn *ldap.Conn
    if u.Scheme == "ldaps" {
        // LDAPS (with TLS)
        tlsConfig := &tls.Config{
            InsecureSkipVerify: c.config.SkipVerify,
        }
        
        if c.config.CACertFile != "" {
            caCert, err := os.ReadFile(c.config.CACertFile)
            if err != nil {
                return nil, err
            }
            certPool := x509.NewCertPool()
            certPool.AppendCertsFromPEM(caCert)
            tlsConfig.RootCAs = certPool
        }
        
        conn, err = ldap.DialTLS("tcp", u.Host, tlsConfig)
    } else {
        // Plain LDAP
        conn, err = ldap.Dial("tcp", u.Host)
        if err != nil {
            return nil, err
        }
        
        // StartTLS if enabled
        if c.config.StartTLS {
            tlsConfig := &tls.Config{InsecureSkipVerify: c.config.SkipVerify}
            if err := conn.StartTLS(tlsConfig); err != nil {
                conn.Close()
                return nil, err
            }
        }
    }
    
    return conn, err
}

func (c *LDAPClient) getUserGroups(conn *ldap.Conn, userDN string) ([]string, error) {
    searchRequest := ldap.NewSearchRequest(
        c.config.GroupBaseDN,
        ldap.ScopeWholeSubtree,
        ldap.NeverDerefAliases,
        0, 0, false,
        strings.Replace(c.config.GroupFilter, "{userdn}", ldap.EscapeFilter(userDN), -1),
        []string{c.config.GroupNameAttribute},
        nil,
    )
    
    searchResult, err := conn.Search(searchRequest)
    if err != nil {
        return nil, err
    }
    
    var groups []string
    for _, entry := range searchResult.Entries {
        groupName := entry.GetAttributeValue(c.config.GroupNameAttribute)
        if groupName != "" {
            groups = append(groups, groupName)
        }
    }
    
    return groups, nil
}

type LDAPUser struct {
    DN       string
    Username string
    Email    string
    FullName string
    Groups   []string
}
```

**2. LDAP Handler**

```go
// internal/api/handlers/ldap.go

type LDAPHandler struct {
    ldapClient        *ldap.LDAPClient
    db                storage.Database
    jwtSvc            *auth.JWTService
    tenantProvisioner *tenant.TenantProvisioner
    logger            *logrus.Logger
}

func (h *LDAPHandler) Login(c *gin.Context) {
    var req struct {
        Username string `json:"username" binding:"required"`
        Password string `json:"password" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
        return
    }
    
    // Authenticate via LDAP
    ldapUser, err := h.ldapClient.Authenticate(req.Username, req.Password)
    if err != nil {
        h.logger.WithError(err).Warnf("LDAP auth failed for user: %s", req.Username)
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
        return
    }
    
    // Create or update user
    user, err := h.provisionUser(c.Request.Context(), ldapUser)
    if err != nil {
        h.logger.WithError(err).Error("Failed to provision user")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "user provisioning failed"})
        return
    }
    
    // Provision tenants from groups (similar to OIDC-02)
    if len(ldapUser.Groups) > 0 {
        mappings := parseGroupMappings(ldapUser.Groups, h.ldapClient.config.GroupToTenant)
        if err := h.tenantProvisioner.ProvisionTenantsForUser(c.Request.Context(), user.ID, mappings); err != nil {
            h.logger.WithError(err).Error("Failed to provision tenants")
        }
    }
    
    // Issue JWT token
    token, err := h.jwtSvc.GenerateToken(user.ID, user.Role)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "token": token,
        "user":  user,
    })
}

func (h *LDAPHandler) provisionUser(ctx context.Context, ldapUser *ldap.LDAPUser) (*models.User, error) {
    // Check if user exists (by LDAP DN or username)
    user, err := h.db.GetUserByLDAPDN(ctx, ldapUser.DN)
    if err == nil {
        // User exists, update if auto-update enabled
        if h.ldapClient.config.AutoUpdateUser {
            user.Email = ldapUser.Email
            user.FullName = ldapUser.FullName
            user.UpdatedAt = time.Now()
            if err := h.db.UpdateUser(ctx, user); err != nil {
                return nil, err
            }
        }
        return user, nil
    }
    
    // User doesn't exist, create if auto-create enabled
    if !h.ldapClient.config.AutoCreateUser {
        return nil, errors.New("user auto-creation disabled")
    }
    
    user = &models.User{
        ID:           uuid.New().String(),
        Username:     ldapUser.Username,
        Email:        ldapUser.Email,
        FullName:     ldapUser.FullName,
        Role:         models.UserRoleUser,
        AuthProvider: models.AuthProviderLDAP,
        LDAPDN:       &ldapUser.DN,
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
    }
    
    if err := h.db.CreateUser(ctx, user); err != nil {
        return nil, err
    }
    
    h.logger.WithFields(logrus.Fields{
        "user_id":  user.ID,
        "username": user.Username,
        "ldap_dn":  ldapUser.DN,
    }).Info("User auto-provisioned via LDAP")
    
    return user, nil
}
```

**3. Database Schema Updates**

```sql
-- Add LDAP fields to users table
ALTER TABLE users ADD COLUMN ldap_dn TEXT UNIQUE; -- LDAP Distinguished Name

CREATE INDEX idx_users_ldap_dn ON users(ldap_dn) WHERE ldap_dn IS NOT NULL;
```

### Frontend Integration

**Login Form with LDAP:**

```html
<!-- web/login.html -->
<div class="login-container">
  <h2>Login</h2>
  
  <!-- LDAP Login (Active Directory credentials) -->
  <form id="ldap-login-form">
    <input type="text" name="username" placeholder="Username" />
    <input type="password" name="password" placeholder="Password" />
    <button type="submit">Login with LDAP</button>
  </form>
  
  <div class="divider">OR</div>
  
  <!-- OIDC Login -->
  <button onclick="loginWithOIDC()">Login with Keycloak</button>
</div>

<script>
document.getElementById('ldap-login-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const formData = new FormData(e.target);
  
  const response = await fetch('/auth/ldap/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      username: formData.get('username'),
      password: formData.get('password')
    })
  });
  
  const { token, user } = await response.json();
  localStorage.setItem('token', token);
  window.location.href = '/dashboard';
});
</script>
```

### Admin UI Configuration

**Admin Panel → Settings → LDAP:**

```html
<div class="ldap-settings">
  <h3>LDAP / Active Directory Configuration</h3>
  
  <label>
    <input type="checkbox" id="ldap-enabled" />
    Enable LDAP Authentication
  </label>
  
  <input type="text" id="ldap-url" placeholder="ldap://ldap.example.com:389" />
  <input type="text" id="ldap-bind-dn" placeholder="cn=admin,dc=example,dc=com" />
  <input type="password" id="ldap-bind-password" placeholder="Bind Password" />
  
  <h4>User Search</h4>
  <input type="text" id="ldap-user-base-dn" placeholder="ou=users,dc=example,dc=com" />
  <input type="text" id="ldap-user-filter" placeholder="(uid={username})" />
  
  <h4>Group Search</h4>
  <input type="text" id="ldap-group-base-dn" placeholder="ou=groups,dc=example,dc=com" />
  <input type="text" id="ldap-group-filter" placeholder="(member={userdn})" />
  
  <label>
    <input type="checkbox" id="ldap-auto-create" />
    Auto-create users on first login
  </label>
  
  <button onclick="testLDAPConnection()">Test Connection</button>
  <button onclick="saveLDAPConfig()">Save Configuration</button>
</div>
```

## Security

**1. Secure Bind Password**
- Store encrypted в БД или environment variable
- Never log passwords

**2. TLS/LDAPS**
- Prefer LDAPS (ldaps://) or StartTLS
- Validate SSL certificates (don't skip_verify в production)

**3. Search Injection Prevention**
- ldap.EscapeFilter() для всех user inputs
- Parameterized filters

## Требования

### Функциональные

1. ✅ LDAP bind authentication
2. ✅ User search и validation
3. ✅ Group search и extraction
4. ✅ Auto-provisioning users from LDAP
5. ✅ Group → Tenant mapping (как OIDC-02)
6. ✅ LDAPS и StartTLS support
7. ✅ Admin UI для LDAP configuration
8. ✅ Test connection функция
9. ✅ Fallback to local auth
10. ✅ Support для Active Directory

### Нефункциональные

1. **Security**
   - TLS encryption для LDAP connections
   - Secure storage для bind credentials
   - LDAP injection prevention

2. **Performance**
   - Auth < 1s
   - Connection pooling

3. **Compatibility**
   - OpenLDAP
   - Active Directory
   - FreeIPA

## Acceptance Criteria

- [ ] User может логиниться с LDAP credentials
- [ ] LDAP bind authentication работает
- [ ] User search корректно находит пользователей
- [ ] Groups извлекаются из LDAP
- [ ] Auto-provisioning создает users
- [ ] Group → Tenant mapping работает
- [ ] LDAPS и StartTLS работают
- [ ] Admin UI позволяет настроить LDAP
- [ ] Test connection проверяет LDAP endpoints
- [ ] Active Directory compatibility
- [ ] Unit tests для LDAP client
- [ ] Integration tests с mock LDAP server

## Риски и зависимости

### Риски

1. **LDAP server unavailable**
   - Mitigation: Fallback to local auth, timeout

2. **Complex LDAP schemas** - разные атрибуты
   - Mitigation: Configurable attribute mapping

3. **Performance** - медленный LDAP server
   - Mitigation: Connection timeout, caching

### Зависимости

1. **go-ldap/ldap** - LDAP client (github.com/go-ldap/ldap/v3)
2. LDAP/AD server - external dependency

## Связанные задачи

- **OIDC-01**: Keycloak SSO - альтернативный auth provider
- **OIDC-02**: Auto-tenant from Groups - схожая логика group sync
- **AUDIT-01**: Enhanced Audit Logging - логирование LDAP events

## Примечания

- Only bind authentication (не Kerberos)
- No automatic scheduled sync (только login-time sync)
- Nested groups - limited support
- Read-only LDAP operations (не modify)

---

**Статус:** 📋 Planned for v1.9.0  
**Последнее обновление:** 2025-10-11

