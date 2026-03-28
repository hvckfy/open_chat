package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

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

/*
generate_entropy()
seed = BIP39(entropy)

private_key = derive(seed)
public_key = derive(private_key)

K = Argon2(seed)

encrypted_private_key = AES(private_key, K)

SEND TO SERVER:

	public_key
	encrypted_private_key

keys is database is BYTEA (byte array)
user sends public_key and ecnrypted key and base64
*/
func SetKeys(c *gin.Context) {
	var req struct {
		PublicKey           string `json:"public_key"`            //base64
		EncryptedPrivateKey string `json:"encrypted_private_key"` //base64
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogWarn(c, "Invalid GetKeys request format", zap.Error(err))
		RespondError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	var u user.User
	if val, exists := c.Get("user"); exists {
		u = val.(user.User)
	} else {
		logger.LogWarn(c, "Empty user", zap.Error(fmt.Errorf("context user could not be empty")))
		RespondError(c, http.StatusBadRequest, "Check auth. Got empty user fields")
	}

	//base64 -> byte array
	pubKeyBytes, err := base64.StdEncoding.DecodeString(req.PublicKey)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Invalid public_key base64")
		return
	}

	if len(pubKeyBytes) < 32 {
		RespondError(c, http.StatusBadRequest, "Invalid public key")
		return
	}

	encryptedPrivKeyBytes, err := base64.StdEncoding.DecodeString(req.EncryptedPrivateKey)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Invalid encrypted_private_key base64")
		return
	}

	if len(encryptedPrivKeyBytes) < 64 {
		RespondError(c, http.StatusBadRequest, "Invalid encrypted private key")
		return
	}

	success, err := messenger.PutKeys(u.App.UserId, encryptedPrivKeyBytes, pubKeyBytes)
	if err != nil {

		if strings.Contains(err.Error(), "duplicate key") {
			RespondError(c, http.StatusConflict, "User already has keys")
			return
		}

		logger.LogError(c, "Sql error", err)
		RespondError(c, http.StatusInternalServerError, "Internal server error")
		return
	}

	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	resp.Success = success
	resp.Message = "Key exported successfuly"

	RespondSuccess(c, http.StatusOK, resp)

}
