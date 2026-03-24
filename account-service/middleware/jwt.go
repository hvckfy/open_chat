package middleware

import (
	"account-service/services/auth/user"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Get the Authorization header
		authHeader := c.GetHeader("Authorization")

		// 2. Validate header format (e.g., "Bearer <token>")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// 3. Extract the token by removing the "Bearer " prefix
		token := strings.TrimPrefix(authHeader, "Bearer ")

		u, err := user.ValidateAccessJwt(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}
		c.Set("user", u)
		c.Next()
	}
}
