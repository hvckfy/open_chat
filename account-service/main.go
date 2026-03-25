package main

import (
	"account-service/services/config"
	"account-service/services/errofy"
	"account-service/services/logger"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize config
	err := config.InitConfig()
	if err != nil {
		fmt.Printf("Failed to initialize config: %v\n", err)
		return
	}
	err = errofy.InitErrors()
	if err != nil {
		fmt.Printf("Failed to initialize errors: %v\n", err)
		return
	}

	err = logger.InitLogger()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		return
	}
	defer logger.Sync()

	logger.Log.Info("Starting account service")
	logger.Log.Info("TEST ERRORS")

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
