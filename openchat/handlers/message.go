package handlers

import (
	"fmt"
	"net/http"

	"openchat/services/auth/user"
	"openchat/services/logger"
	"openchat/services/messenger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewChat(c *gin.Context) {

	var req struct {
		TargetUsername string `json:"target_username"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogWarn(c, "Invalid new chat request format", zap.Error(err))
		RespondError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// TODO: реализация создания чата

	body := struct {
		Success bool `json:"success"`
	}{
		Success: true,
	}

	RespondSuccess(c, http.StatusOK, body)
}

func GetKey(c *gin.Context) {

	var req struct {
		Words []string `json:"words"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogWarn(c, "Invalid GetKeys request format", zap.Error(err))
		RespondError(c, http.StatusBadRequest, "Invalid request format")
		return
	}
	//should return
}

func GenKeys(c *gin.Context) {

	var u user.User
	if val, exists := c.Get("user"); exists {
		u = val.(user.User)
	} else {
		logger.LogWarn(c, "Empty user", zap.Error(fmt.Errorf("context user could not be empty")))
		RespondError(c, http.StatusBadRequest, "Check auth. Got empty user fields")
		return
	}

	// check if keys already exist
	logger.Info(fmt.Sprintf("Checking keys for user ID: %d", u.App.UserId))
	_, _, exists, err := messenger.GetKeys(u.App.UserId)

	if err != nil {
		logger.LogError(c, "Bad response for getting keys from db", err)
		RespondError(c, http.StatusInternalServerError, "Internal server error")
		return
	}

	logger.Info(fmt.Sprintf("Keys exist for user %d: %t", u.App.UserId, exists))
	if exists {
		logger.LogWarn(c, "You already have keys", zap.Error(fmt.Errorf("user doesnt allowed to have two keychains")))
		RespondError(c, http.StatusBadRequest, "User already has keys")
		return
	}
	logger.Info(fmt.Sprintf("Generating noew keys for user: %v", u))
	fmt.Printf("Generating noew keys for user: %v", u)
	// generate keys
	words, privKey, err := messenger.GenKeys(u.App.UserId)
	if err != nil {
		logger.LogError(c, "GenKeys error", err)
		RespondError(c, http.StatusInternalServerError, "GenKeys failed")
		return
	}

	body := struct {
		Words      []string `json:"words"`
		PrivateKey string   `json:"private_key"`
	}{
		Words:      words,
		PrivateKey: privKey,
	}
	//user have to storage PrivateKey and decrypt all incomming messages by them.
	//data base has Public Key that other users use to encrypt their messages to this user
	RespondSuccess(c, http.StatusOK, body)
}
