package messagemiddleware

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"openchat/handlers"
	"openchat/services/config"
	"os/user"

	"github.com/gin-gonic/gin"
)

func CookieAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("access_token")
		if err != nil {
			c.JSON(401, gin.H{"error": "No access cookie"})
			c.Abort()
			return
		}
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: config.Data.Mtls,
			},
		}
		payload := map[string]string{
			"access_token": token,
		}
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			c.JSON(401, gin.H{"error": "Bad access token"})
			c.Abort()
			return
		}
		url := "account-service:48080/api/account/service/verify-access-token"
		resp, err := client.Post(url, "application/json", bytes.NewReader(jsonPayload))
		if err != nil {
			log.Printf("cant access mtls host - account-service: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			c.Abort()
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("cant read resp.body: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			c.Abort()
			return
		}
		// Parse response
		var serviceResp handlers.ServiceResponse
		if err := json.Unmarshal(body, &serviceResp); err != nil {
			log.Printf("cant parse response: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			c.Abort()
			return
		}

		if serviceResp.Err.Exists == "true" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": serviceResp.Err.Message})
			return
		}

		// Распарсим user.User из поля response
		var user user.User
		if err := json.Unmarshal(serviceResp.Response, &user); err != nil {
			panic(err)
		}
		c.Set("user", user)
		c.Next()
	}
}
