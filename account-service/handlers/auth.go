package handlers

import (
	"account-service/services/auth/ldap"
	"account-service/services/auth/user"
	"account-service/services/errofy"

	"github.com/gin-gonic/gin"
)

func LdapLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errofy.LogError(495, err, "LdapLogin")
		status, response := errofy.RaiseApiLogic(495)
		c.JSON(status, response)
		return
	}
	access, refresh, errorCode, err := ldap.AuthUser(req.Username, req.Password)
	if err != nil {
		status, response := errofy.RaiseApiLogic(errorCode)
		c.JSON(status, response)
		return
	}
	status, _ := errofy.RaiseApiLogic(errorCode)
	c.JSON(status, gin.H{"access_token": access, "refresh_token": refresh})
}

func NoLdapLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errofy.LogError(495, err, "NoLdapLogin")
		status, response := errofy.RaiseApiLogic(495)
		c.JSON(status, response)
		return
	}
	access, refresh, errorCode, err := ldap.AuthUser(req.Username, req.Password)
	if err != nil {
		status, response := errofy.RaiseApiLogic(errorCode)
		c.JSON(status, response)
		return
	}
	status, _ := errofy.RaiseApiLogic(errorCode)
	c.JSON(status, gin.H{"access_token": access, "refresh_token": refresh})
}

func RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(errofy.RaiseApiLogic(400))
		return
	}
	access, err := user.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(errofy.RaiseApiLogic(401))
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
		c.JSON(errofy.RaiseApiLogic(400))
		return
	}
	success, err := user.TerminateToken(req.RefreshToken)
	if err != nil {
		c.JSON(errofy.RaiseApiLogic(500))
		return
	}
	c.JSON(200, gin.H{"message": success})
}
