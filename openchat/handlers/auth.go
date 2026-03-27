package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"openchat/services/auth/ldap"
	"openchat/services/auth/local"
	"openchat/services/auth/user"
	"openchat/services/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func LdapLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return
	}

	refresh, access, err := ldap.AuthUser(req.Username, req.Password)
	if err != nil {
		log.Printf("ERROR LdapLogin: authentication failed for %s: %v", req.Username, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	// Set short-lived access cookie (15min)
	c.SetCookie("access_token", access, 15*60, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"refresh_token": refresh,
		"expires_in":    900, // 15min
	})
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

	// Set short-lived access cookie (15min)
	c.SetCookie("access_token", access, 15*60, "/", "", false, true)

	logger.LogAuthSuccess(c, req.Username, "login")
	c.JSON(http.StatusOK, gin.H{
		"refresh_token": refresh,
		"expires_in":    900, // 15min
	})
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

	// Set short-lived access cookie (15min)
	c.SetCookie("access_token", access, 15*60, "/", "", false, true)

	c.JSON(http.StatusCreated, gin.H{
		"refresh_token": refresh,
		"expires_in":    900, // 15min
	})
}

func RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ERROR RefreshToken: invalid format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	access, err := user.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		log.Printf("ERROR RefreshToken: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Set new access cookie (15min)
	c.SetCookie("access_token", access, 15*60, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"expires_in": 900})
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
	if err != nil || !success {
		log.Printf("ERROR RevokeToken: failed to revoke token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": success})
}

func RevokeAllTokens(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ERROR RevokeToken: invalid request format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	success, err := user.TerminateAll(req.RefreshToken)
	if err != nil || !success {
		log.Printf("ERROR RevokeToken: failed to revoke token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": success})
}

func ServiceAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			AccessToken string `json:"access_token"`
		}
		err := c.ShouldBindJSON(&req)
		if err != nil {
			log.Printf("ERROR RevokeToken: invalid request format: %v", err)
			resp := ServiceResponse{
				Err: ServiceResponseError{
					Exists:  "true",
					Message: "Bad Request",
				},
			}
			c.JSON(http.StatusBadRequest, resp)
			return
		}
		u, err := user.ValidateAccessJwt(req.AccessToken)
		if err != nil {
			log.Printf("CookieAuthMiddleware invalid token: %v", err)
			resp := ServiceResponse{
				Err: ServiceResponseError{
					Exists:  "true",
					Message: "Invalid access token",
				},
			}
			c.JSON(http.StatusUnauthorized, resp)
			return
		}
		JsonUser, err := json.Marshal(u)
		if err != nil {
			log.Printf("CookieAuthMiddleware invalid token: %v", err)
			resp := ServiceResponse{
				Err: ServiceResponseError{
					Exists:  "true",
					Message: "Cant marshal response",
				},
			}
			c.JSON(http.StatusInternalServerError, resp)
			return
		}
		resp := ServiceResponse{
			Response: JsonUser,
			Err: ServiceResponseError{
				Exists: "false",
			},
		}
		c.JSON(http.StatusOK, resp)
	}
}
