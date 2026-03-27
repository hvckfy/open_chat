package handlers

import (
	"log"
	"net/http"

	"openchat/services/auth/ldap"
	"openchat/services/auth/local"
	"openchat/services/auth/user"
	"openchat/services/config"
	"openchat/services/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
Handler for ldap auth.
*/
func LdapLogin(c *gin.Context) {

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogWarn(c, "Invalid LDAP login request format", zap.Error(err))
		RespondError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	refresh, access, err := ldap.AuthUser(req.Username, req.Password)
	if err != nil {

		logger.LogWarn(c, "LDAP authentication failed",
			zap.String("username", req.Username),
			zap.Error(err))

		RespondError(c, http.StatusUnauthorized, "Authentication failed")
		return
	}

	body := struct {
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}{
		RefreshToken: refresh,
		ExpiresIn:    config.Data.JWT.RefreshTokenExpire,
	}

	c.SetCookie(
		"access_token",
		access,
		int(config.Data.JWT.AccessTokenExpire),
		"/",
		"",
		false,
		true,
	)

	RespondSuccess(c, http.StatusOK, body)
}

/*
Handler for openchat auth.
*/
func LocalLogin(c *gin.Context) {

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {

		logger.LogWarn(c, "Invalid login request format", zap.Error(err))
		RespondError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	refresh, access, err := local.AuthUser(req.Username, req.Password)
	if err != nil {

		logger.LogWarn(c, "Authentication failed",
			zap.String("username", req.Username),
			zap.Error(err))

		RespondError(c, http.StatusUnauthorized, "Authentication failed")
		return
	}

	body := struct {
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}{
		RefreshToken: refresh,
		ExpiresIn:    config.Data.JWT.RefreshTokenExpire,
	}

	c.SetCookie(
		"access_token",
		access,
		int(config.Data.JWT.AccessTokenExpire),
		"/",
		"",
		false,
		true,
	)

	RespondSuccess(c, http.StatusOK, body)
}

func LocalRegister(c *gin.Context) {

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {

		logger.LogWarn(c, "Invalid registration request format", zap.Error(err))
		RespondError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	refresh, access, err := local.RegisterUser(req.Username, req.Password)
	if err != nil {

		logger.LogWarn(c, "User registration failed",
			zap.String("username", req.Username),
			zap.Error(err))

		RespondError(c, http.StatusConflict, "Registration failed")
		return
	}

	body := struct {
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}{
		RefreshToken: refresh,
		ExpiresIn:    config.Data.JWT.RefreshTokenExpire,
	}

	c.SetCookie(
		"access_token",
		access,
		int(config.Data.JWT.AccessTokenExpire),
		"/",
		"",
		false,
		true,
	)

	RespondSuccess(c, http.StatusOK, body)
}

func RefreshToken(c *gin.Context) {

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {

		log.Printf("ERROR RefreshToken: invalid format: %v", err)
		RespondError(c, http.StatusBadRequest, "Provided invalid format")
		return
	}

	access, err := user.ValidateRefreshToken(req.RefreshToken)
	if err != nil {

		log.Printf("ERROR RefreshToken: %v", err)
		RespondError(c, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	body := struct {
		Success   bool  `json:"success"`
		ExpiresIn int64 `json:"expires_in"`
	}{
		Success:   true,
		ExpiresIn: config.Data.JWT.AccessTokenExpire,
	}

	c.SetCookie(
		"access_token",
		access,
		int(config.Data.JWT.AccessTokenExpire),
		"/",
		"",
		false,
		true,
	)

	RespondSuccess(c, http.StatusOK, body)
}

func Profile(c *gin.Context) {

	u_, exists := c.Get("user")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "Unauthorized")
	}
	u := u_.(user.User)

	RespondSuccess(c, http.StatusOK, u)
}

func RevokeToken(c *gin.Context) {

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {

		log.Printf("ERROR RevokeToken: invalid request format: %v", err)
		RespondError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	success, err := user.TerminateToken(req.RefreshToken)
	if err != nil || !success {

		log.Printf("ERROR RevokeToken: failed to revoke token: %v", err)
		RespondError(c, http.StatusInternalServerError, "Failed to revoke token")
		return
	}

	body := struct {
		Success bool `json:"success"`
	}{
		Success: true,
	}

	RespondSuccess(c, http.StatusOK, body)
}

func RevokeAllTokens(c *gin.Context) {

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {

		log.Printf("ERROR RevokeAllTokens: invalid request format: %v", err)
		RespondError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	success, err := user.TerminateAll(req.RefreshToken)
	if err != nil || !success {

		log.Printf("ERROR RevokeAllTokens: failed to revoke tokens: %v", err)
		RespondError(c, http.StatusInternalServerError, "Failed to revoke tokens")
		return
	}

	body := struct {
		Success bool `json:"success"`
	}{
		Success: true,
	}

	RespondSuccess(c, http.StatusOK, body)
}

func ServiceAuth() gin.HandlerFunc {

	return func(c *gin.Context) {

		var req struct {
			AccessToken string `json:"access_token"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {

			log.Printf("ERROR ServiceAuth: invalid request format: %v", err)
			RespondError(c, http.StatusBadRequest, "Bad Request")
			return
		}

		u, err := user.ValidateAccessJwt(req.AccessToken)
		if err != nil {

			log.Printf("ServiceAuth invalid token: %v", err)
			RespondError(c, http.StatusUnauthorized, "Invalid access token")
			return
		}

		RespondSuccess(c, http.StatusOK, u)
	}
}
