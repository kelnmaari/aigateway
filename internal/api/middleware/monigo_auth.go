package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"ollama-openai-proxy/internal/auth/jwt"
	"ollama-openai-proxy/internal/storage"
)

// MonigoJWTAuth создает middleware для защиты MoniGo dashboard через JWT
// Только admin users могут получить доступ к performance monitoring
func MonigoJWTAuth(jwtManager *jwt.Manager, db storage.Database) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Static files и assets должны пропускаться без аутентификации
			if isStaticFile(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Попытка извлечь JWT токен из cookie
			token := extractTokenFromCookie(r)
			if token == "" {
				// Если токен не найден в cookie, пробуем Authorization header
				token = extractTokenFromAuthHeader(r)
			}

			if token == "" {
				http.Error(w, "Unauthorized: Missing JWT token", http.StatusUnauthorized)
				return
			}

			// Валидация токена
			claims, err := jwtManager.ValidateAccessToken(token)
			if err != nil {
				http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
				return
			}

			// Получаем пользователя из БД для проверки admin прав
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			user, err := db.GetUser(ctx, claims.UserID)
			if err != nil {
				http.Error(w, "Unauthorized: User not found", http.StatusUnauthorized)
				return
			}

			// Проверка admin прав
			if !user.IsAdmin {
				http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
				return
			}

			// Пользователь авторизован и является admin
			next.ServeHTTP(w, r)
		})
	}
}

// isStaticFile проверяет, является ли запрос к статическому файлу
// MoniGo dashboard использует статические CSS/JS файлы
func isStaticFile(path string) bool {
	staticExtensions := []string{
		".css", ".js", ".ico", ".png", ".jpg", ".jpeg", ".gif", ".svg",
		".woff", ".woff2", ".ttf", ".eot", ".map",
	}

	pathLower := strings.ToLower(path)
	for _, ext := range staticExtensions {
		if strings.HasSuffix(pathLower, ext) {
			return true
		}
	}

	return false
}

// extractTokenFromCookie извлекает JWT токен из cookie "access_token"
func extractTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie("access_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// extractTokenFromAuthHeader извлекает JWT токен из Authorization header
func extractTokenFromAuthHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Формат: "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}
