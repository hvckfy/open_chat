package middleware

import (
	"log"
	"openchat/services/auth/user"

	"github.com/gin-gonic/gin"
)

func CookieAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("access_token")
		if err != nil {
			c.JSON(401, gin.H{"error": "No access cookie"})
			c.Abort()
			return
		}

		u, err := user.ValidateAccessJwt(token)
		if err != nil {
			log.Printf("CookieAuthMiddleware invalid token: %v", err)
			c.JSON(401, gin.H{"error": "Invalid access token"})
			c.Abort()
			return
		}
		c.Set("user", u)
		c.Next()
	}
}
