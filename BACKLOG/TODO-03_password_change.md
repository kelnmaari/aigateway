# TODO-03: Password Change Implementation

**Версия:** v1.5.2  
**Приоритет:** MEDIUM  
**Оценка:** 3-4 часа  
**Зависимости:** Нет  
**Статус:** 📋 Planned  
**Тип:** Feature Implementation

---

## 📋 Описание

Реализовать backend для смены пароля. UI уже существует в `web/profile.html`, но endpoint возвращает "Not implemented".

## 📊 Current Status

### Frontend (Уже готов)

**Файл:** `web/profile.html`

```html
<div class="card">
    <div class="card-header">
        <h5>Change Password</h5>
    </div>
    <div class="card-body">
        <form id="changePasswordForm">
            <div class="mb-3">
                <label class="form-label">Current Password</label>
                <input type="password" class="form-control" id="currentPassword" required>
            </div>
            <div class="mb-3">
                <label class="form-label">New Password</label>
                <input type="password" class="form-control" id="newPassword" required>
            </div>
            <div class="mb-3">
                <label class="form-label">Confirm New Password</label>
                <input type="password" class="form-control" id="confirmPassword" required>
            </div>
            <button type="submit" class="btn btn-primary">Change Password</button>
        </form>
    </div>
</div>
```

**Файл:** `web/js/profile.js`

```javascript
// Frontend уже делает запрос
document.getElementById('changePasswordForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const currentPassword = document.getElementById('currentPassword').value;
    const newPassword = document.getElementById('newPassword').value;
    const confirmPassword = document.getElementById('confirmPassword').value;
    
    if (newPassword !== confirmPassword) {
        showError('Passwords do not match');
        return;
    }
    
    const response = await fetch('/api/auth/change-password', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${getToken()}`
        },
        body: JSON.stringify({
            current_password: currentPassword,
            new_password: newPassword
        })
    });
    
    // ... handle response
});
```

### Backend (TODO)

**Файл:** `internal/api/handlers/auth.go` (строка 203)

```go
// ChangePassword изменяет пароль пользователя
func (h *AuthHandlers) ChangePassword(c *gin.Context) {
    // TODO: Implement password change
    c.JSON(http.StatusNotImplemented, gin.H{
        "error": "Password change not implemented yet",
    })
}
```

## 🔧 Implementation

### 1. Request/Response Models

**Файл:** `internal/models/auth.go`

```go
// ChangePasswordRequest запрос на смену пароля
type ChangePasswordRequest struct {
    CurrentPassword string `json:"current_password" binding:"required,min=8"`
    NewPassword     string `json:"new_password" binding:"required,min=8,max=100"`
}

// ChangePasswordResponse ответ на смену пароля
type ChangePasswordResponse struct {
    Message string `json:"message"`
}
```

### 2. Handler Implementation

**Файл:** `internal/api/handlers/auth.go`

```go
import (
    "net/http"
    "github.com/gin-gonic/gin"
    "golang.org/x/crypto/bcrypt"
)

// ChangePassword изменяет пароль пользователя
func (h *AuthHandlers) ChangePassword(c *gin.Context) {
    // Get user ID from JWT middleware
    userID := c.GetString("user_id")
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }
    
    // Parse request
    var req models.ChangePasswordRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid request",
            "details": err.Error(),
        })
        return
    }
    
    // Get user from database
    user, err := h.userStorage.GetUserByID(c.Request.Context(), userID)
    if err != nil {
        h.logger.WithError(err).Error("Failed to get user")
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to get user",
        })
        return
    }
    
    // Verify current password
    if err := bcrypt.CompareHashAndPassword(
        []byte(user.PasswordHash), 
        []byte(req.CurrentPassword),
    ); err != nil {
        h.logger.WithField("user_id", userID).Warn("Invalid current password")
        c.JSON(http.StatusUnauthorized, gin.H{
            "error": "Current password is incorrect",
        })
        return
    }
    
    // Validate new password strength
    if err := validatePasswordStrength(req.NewPassword); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "New password does not meet requirements",
            "details": err.Error(),
        })
        return
    }
    
    // Check if new password is same as current
    if req.CurrentPassword == req.NewPassword {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "New password must be different from current password",
        })
        return
    }
    
    // Hash new password
    newHash, err := bcrypt.GenerateFromPassword(
        []byte(req.NewPassword), 
        bcrypt.DefaultCost,
    )
    if err != nil {
        h.logger.WithError(err).Error("Failed to hash password")
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to change password",
        })
        return
    }
    
    // Update password in database
    if err := h.userStorage.UpdateUserPassword(
        c.Request.Context(), 
        userID, 
        string(newHash),
    ); err != nil {
        h.logger.WithError(err).Error("Failed to update password")
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to change password",
        })
        return
    }
    
    // Log the password change
    h.logger.WithField("user_id", userID).Info("Password changed successfully")
    
    // Optionally: invalidate all existing sessions except current
    // h.jwtService.InvalidateUserSessions(userID, getCurrentToken(c))
    
    c.JSON(http.StatusOK, models.ChangePasswordResponse{
        Message: "Password changed successfully",
    })
}

// validatePasswordStrength проверяет надежность пароля
func validatePasswordStrength(password string) error {
    if len(password) < 8 {
        return fmt.Errorf("password must be at least 8 characters long")
    }
    
    if len(password) > 100 {
        return fmt.Errorf("password must be less than 100 characters")
    }
    
    // Check for at least one uppercase letter
    hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
    if !hasUpper {
        return fmt.Errorf("password must contain at least one uppercase letter")
    }
    
    // Check for at least one lowercase letter
    hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
    if !hasLower {
        return fmt.Errorf("password must contain at least one lowercase letter")
    }
    
    // Check for at least one digit
    hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
    if !hasDigit {
        return fmt.Errorf("password must contain at least one digit")
    }
    
    // Optional: Check for special character
    // hasSpecial := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password)
    // if !hasSpecial {
    //     return fmt.Errorf("password must contain at least one special character")
    // }
    
    return nil
}
```

### 3. Storage Layer

**Файл:** `internal/storage/user.go`

Добавить метод (если отсутствует):

```go
// UpdateUserPassword обновляет пароль пользователя
UpdateUserPassword(ctx context.Context, userID string, passwordHash string) error
```

**Реализация для SQLite:**

**Файл:** `internal/storage/sqlite/users.go`

```go
func (s *SQLiteStorage) UpdateUserPassword(ctx context.Context, userID string, passwordHash string) error {
    query := `
        UPDATE users 
        SET password_hash = ?, updated_at = CURRENT_TIMESTAMP 
        WHERE id = ?
    `
    
    result, err := s.db.ExecContext(ctx, query, passwordHash, userID)
    if err != nil {
        return fmt.Errorf("failed to update password: %w", err)
    }
    
    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    
    if rows == 0 {
        return fmt.Errorf("user not found")
    }
    
    return nil
}
```

**Реализация для PostgreSQL:**

**Файл:** `internal/storage/postgresql/users.go`

```go
func (s *PostgresStorage) UpdateUserPassword(ctx context.Context, userID string, passwordHash string) error {
    query := `
        UPDATE users 
        SET password_hash = $1, updated_at = NOW() 
        WHERE id = $2
    `
    
    result, err := s.db.ExecContext(ctx, query, passwordHash, userID)
    if err != nil {
        return fmt.Errorf("failed to update password: %w", err)
    }
    
    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    
    if rows == 0 {
        return fmt.Errorf("user not found")
    }
    
    return nil
}
```

### 4. Router (Уже должен быть)

**Файл:** `internal/api/router/router.go`

```go
// В setupAuthRoutes():
auth.POST("/change-password", authHandlers.ChangePassword) // Проверить наличие
```

## 📝 Implementation Plan

1. **Update models** (15 мин)
   - ChangePasswordRequest
   - ChangePasswordResponse

2. **Implement storage methods** (30 мин)
   - UpdateUserPassword для SQLite
   - UpdateUserPassword для PostgreSQL

3. **Implement handler** (1 час)
   - Validate current password
   - Validate new password strength
   - Hash and update

4. **Testing** (1 час)
   - Unit tests для handler
   - Integration tests
   - Manual testing через UI

5. **Security enhancements** (30 мин)
   - Audit logging
   - Session invalidation (optional)
   - Rate limiting (optional)

## ✅ Acceptance Criteria

- [ ] Endpoint `/api/auth/change-password` реализован
- [ ] Валидация текущего пароля работает
- [ ] Валидация нового пароля (сила пароля)
- [ ] Новый пароль сохраняется с bcrypt hash
- [ ] Проверка на совпадение нового и старого пароля
- [ ] Error handling корректный
- [ ] Audit logging добавлен
- [ ] UI отображает success/error сообщения
- [ ] Unit tests написаны

## 🧪 Testing

### Unit Tests

```go
func TestChangePassword(t *testing.T) {
    // Test cases:
    // - Success case
    // - Invalid current password
    // - Weak new password
    // - Same as current password
    // - User not found
}
```

### API Testing

```bash
# Change password
curl -X POST http://localhost:8080/api/auth/change-password \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "current_password": "OldPass123",
    "new_password": "NewSecurePass456"
  }'

# Expected response:
{"message": "Password changed successfully"}
```

### Security Testing

- [ ] Старый пароль не работает после смены
- [ ] Новый пароль работает
- [ ] Rate limiting на endpoint (опционально)

## 🔒 Security Considerations

1. **Rate Limiting** - добавить ограничение попыток
2. **Audit Log** - логировать все смены пароля
3. **Session Invalidation** - опционально убить другие сессии
4. **Password History** - опционально запретить старые пароли
5. **Notification** - опционально отправить email о смене

## 📚 References

- [bcrypt package](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
- [OWASP Password Guidelines](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html#password-complexity)
