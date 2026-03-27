package accountmiddleware

import (
	"log"
	"net/http"
	"openchat/services/auth/user"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")

		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			respondError(c, http.StatusUnauthorized, "Unauthorized")
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		u, err := user.ValidateAccessJwt(token)
		if err != nil {
			log.Printf("ERROR AuthMiddleware: invalid token: %v", err)
			respondError(c, http.StatusUnauthorized, "Unauthorized")
			return
		}

		c.Set("user", u)

		c.Next()
	}
}
