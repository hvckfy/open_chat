package middleware

import (
	"account-service/services/auth/user"
	"account-service/services/errofy"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Get the Authorization header
		authHeader := c.GetHeader("Authorization")

		// 2. Validate header format (e.g., "Bearer <token>")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(errofy.RaiseApiLogic(401))
			c.Abort()
			return
		}

		// 3. Extract the token by removing the "Bearer " prefix
		token := strings.TrimPrefix(authHeader, "Bearer ")

		u, errorCode, err := user.ValidateAccessJwt(token)
		if err != nil {
			status, response := errofy.RaiseApiLogic(errorCode)
			c.JSON(status, response)
			c.Abort()
			return
		}
		c.Set("user", u)
		c.Next()
	}
}
