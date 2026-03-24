package middleware

import (
	"account-service/services/config"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ExtRegPermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.Data.ExternalAllowReg == true {
			c.Set("External", true)
			c.Next()
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "External registartion is not allowed"})
			c.Abort()
			return
		}
	}
}

func ExtRegCheckCode() gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.GetHeader("RegCode") // или c.Query("reg_code")
		if code == config.Data.ExternalRegCode || config.Data.ExternalRegCode == "" {
			c.Set("RegAllowed", true)
			c.Next()
		} else {
			c.AbortWithStatusJSON(403, gin.H{"error": "Invalid reg code"})
		}
	}
}
