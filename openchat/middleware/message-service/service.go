package messagemiddleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"openchat/handlers"
	"openchat/services/auth/user"
	"openchat/services/config"

	"github.com/gin-gonic/gin"
)

func CookieAuthMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		token, err := c.Cookie("access_token")
		if err != nil {
			handlers.RespondError(c, http.StatusUnauthorized, "No access cookie")
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
			handlers.RespondError(c, http.StatusUnauthorized, "No access cookie")
			c.Abort()
			return
		}

		url := "https://account-service:48080/api/account/service/verify-access-token"

		resp, err := client.Post(
			url,
			"application/json",
			bytes.NewReader(jsonPayload),
		)

		if err != nil {
			log.Printf("cant access mtls host - account-service: %v", err)
			handlers.RespondError(c, http.StatusUnauthorized, "No access cookie")
			c.Abort()
			return
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("cant read resp.body: %v", err)
			handlers.RespondError(c, http.StatusUnauthorized, "No access cookie")
			c.Abort()
			return
		}

		var serviceResp handlers.ServiceResponse

		if err := json.Unmarshal(body, &serviceResp); err != nil {
			log.Printf("cant parse response: %v", err)
			handlers.RespondError(c, http.StatusUnauthorized, "No access cookie")
			c.Abort()
			return
		}

		if serviceResp.Err.Exists == "true" {
			handlers.RespondError(c, http.StatusUnauthorized, serviceResp.Err.Message)
			c.Abort()
			return
		}

		var u user.User

		if err := json.Unmarshal([]byte(serviceResp.Response), &u); err != nil {
			log.Printf("cant unmarshal user: %v", err)
			handlers.RespondError(c, http.StatusUnauthorized, "No access cookie")
			c.Abort()
			return
		}

		c.Set("user", u)

		c.Next()
	}
}
