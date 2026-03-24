package main

import (
	"account-service/services/auth/ldap"
	"account-service/services/config"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	err := config.InitConfig()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(ldap.AuthUser("rmiftakhov", "Belayaakula2001-"))

	router := gin.Default()

	router.POST("/login", func(c *gin.Context) {
		c.String(http.StatusAccepted, "OK")
	})

	router.POST("/refresh-token", func(c *gin.Context) {
		c.String(http.StatusAccepted, "OK")
	})

	router.GET("/profile", func(c *gin.Context) {
		c.String(http.StatusAccepted, "OK")
	})

	//router.DELETE("/revoke-token"), func(c *gin.Context)
	//
	//	})
}
