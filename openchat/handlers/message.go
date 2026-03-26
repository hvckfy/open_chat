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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

}

func GetKey(c *gin.Context) {
	var req struct {
		Words []string `json:"words"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogWarn(c, "Invalid GetKeys request format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

}

func GenKeys(c *gin.Context) {
	var u user.User
	if val, exists := c.Get("user"); exists {
		u = val.(user.User)
	} else {
		logger.LogWarn(c, "Empty user", zap.Error(fmt.Errorf("context user could not be empty")))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Check auth. Got empty user fields"})
		return
	}
	//call for function to check if user have keys -> if has then deny
	_, _, exists, err := messenger.GetKeys(u.App.UserId)
	if err != nil {
		logger.LogWarn(c, "Bad response for getting keys from db", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	if exists {
		logger.LogWarn(c, "You already have keys", zap.Error(fmt.Errorf("user doesnt allowed to have two keychains")))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}
	//call for function to genkeys and put into db
	words, priv_key, err := messenger.GenKeys(u.App.UserId)
	if err != nil {
		logger.LogError(c, "GenKeys error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GenKeys failed"})
		return
	}
	//return to front
	c.JSON(http.StatusOK, gin.H{"words": words, "private_key": priv_key})
}
