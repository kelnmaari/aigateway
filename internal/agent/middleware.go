package agent

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth returns a gin middleware that validates Bearer token against expected key.
func APIKeyAuth(expectedKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			auth = c.GetHeader("X-API-Key")
		} else {
			auth = strings.TrimPrefix(auth, "Bearer ")
		}

		if auth == "" || subtle.ConstantTimeCompare([]byte(auth), []byte(expectedKey)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized: invalid or missing API key",
			})
			return
		}
		c.Next()
	}
}
