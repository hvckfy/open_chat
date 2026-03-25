package handlers

import (
	"account-service/services/auth/ldap"
	"account-service/services/auth/local"
	"account-service/services/auth/user"
	"account-service/services/logger"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func LdapLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ERROR LdapLogin: invalid request format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	refresh, access, err := ldap.AuthUser(req.Username, req.Password)
	if err != nil {
		log.Printf("ERROR LdapLogin: authentication failed for %s: %v", req.Username, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": access, "refresh_token": refresh})
}

func LocalLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogWarn(c, "Invalid login request format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	refresh, access, err := local.AuthUser(req.Username, req.Password)
	if err != nil {
		logger.LogWarn(c, "Authentication failed",
			zap.String("username", req.Username),
			zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	logger.LogAuthSuccess(c, req.Username, "login")
	c.JSON(http.StatusOK, gin.H{"access_token": access, "refresh_token": refresh})
}

func LocalRegister(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogWarn(c, "Invalid registration request format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	refresh, access, err := local.RegisterUser(req.Username, req.Password)
	if err != nil {
		logger.LogWarn(c, "User registration failed",
			zap.String("username", req.Username),
			zap.Error(err))
		c.JSON(http.StatusConflict, gin.H{"error": "Registration failed"})
		return
	}

	logger.LogAuthSuccess(c, req.Username, "register")
	c.JSON(http.StatusCreated, gin.H{"access_token": access, "refresh_token": refresh})
}

func RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ERROR RefreshToken: invalid request format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	access, err := user.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		log.Printf("ERROR RefreshToken: invalid refresh token: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": access})
}

func Profile(c *gin.Context) {
	user := c.MustGet("user").(user.User)
	c.JSON(http.StatusOK, user)
}

func RevokeToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ERROR RevokeToken: invalid request format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	success, err := user.TerminateToken(req.RefreshToken)
	if err != nil {
		log.Printf("ERROR RevokeToken: failed to revoke token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": success})
}
