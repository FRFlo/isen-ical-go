package middleware

import (
	"crypto/subtle"
	"strings"

	"github.com/gin-gonic/gin"
)

// AdminAuthMiddleware creates a middleware that validates an admin token.
func AdminAuthMiddleware(adminToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Header("WWW-Authenticate", "Bearer")
			c.AbortWithStatusJSON(401, gin.H{"error": "Authorization header required"})
			return
		}

		// Split "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Header("WWW-Authenticate", "Bearer")
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid authorization format"})
			return
		}

		token := parts[1]
		if token == "" {
			c.Header("WWW-Authenticate", "Bearer")
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid token"})
			return
		}

		// Constant time compare to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(token), []byte(adminToken)) != 1 {
			c.Header("WWW-Authenticate", "Bearer")
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid token"})
			return
		}

		c.Next()
	}
}
