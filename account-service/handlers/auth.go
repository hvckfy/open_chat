package handlers

import (
	"account-service/services/auth/ldap"
	"account-service/services/auth/user"

	"github.com/gin-gonic/gin"
)

func LdapLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	access, refresh, err := ldap.AuthUser(req.Username, req.Password)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"access_token": access, "refresh_token": refresh})
}

func NoLdapLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	access, refresh, err := ldap.AuthUser(req.Username, req.Password)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"access_token": access, "refresh_token": refresh})
}

func RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	access, err := user.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"access_token": access})
}

func Profile(c *gin.Context) {
	user := c.MustGet("user").(user.User)
	c.JSON(200, user)
}

func RevokeToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	success, err := user.TerminateToken(req.RefreshToken)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": success})
}
