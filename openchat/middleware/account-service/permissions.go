package accountmiddleware

import (
	"net/http"
	"openchat/services/config"

	"github.com/gin-gonic/gin"
)

func ExtRegPermission() gin.HandlerFunc {

	return func(c *gin.Context) {

		if config.Data.ExternalAllowReg {
			c.Set("External", true)
			c.Next()
			return
		}

		respondError(c, http.StatusForbidden, "External registration not allowed")
	}
}

func ExtRegCheckCode() gin.HandlerFunc {

	return func(c *gin.Context) {

		code := c.GetHeader("RegCode")

		if code == config.Data.ExternalRegCode || config.Data.ExternalRegCode == "" {
			c.Set("RegAllowed", true)
			c.Next()
			return
		}

		respondError(c, http.StatusForbidden, "Invalid registration code")
	}
}
