package accountmiddleware

import (
	"net/http"
	"openchat/handlers"
	"openchat/services/auth/user"

	"github.com/gin-gonic/gin"
)

func respondError(c *gin.Context, status int, message string) {
	resp := handlers.ServiceResponse{
		Err: handlers.ServiceResponseError{
			Exists:  "true",
			Message: message,
		},
	}

	c.JSON(status, resp)
	c.Abort()
}

func CookieAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		token, err := c.Cookie("access_token")
		if err != nil {
			respondError(c, http.StatusUnauthorized, "No access cookie")
			return
		}

		u, err := user.ValidateAccessJwt(token)
		if err != nil {
			respondError(c, http.StatusUnauthorized, "Invalid access token")
			return
		}

		c.Set("user", u)
		c.Next()
	}
}
