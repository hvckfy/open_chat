package main

import (
	"log"
	"net/http"
	"openchat/services/config"
	"openchat/services/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// Initialize config
	err := config.InitMessageServiceConfig()
	if err != nil {
		log.Printf("ERROR: Failed to initialize config: %v", err)
		return
	}

	// Initialize logger
	err = logger.InitLogger()
	if err != nil {
		log.Printf("ERROR: Failed to initialize logger: %v", err)
		return
	}
	defer logger.Sync()

	logger.Info("Starting message service",
		zap.String("port", "8080"))

	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	port := config.Data.Service.Port
	logger.Info("Server starting",
		zap.String("port", port),
		zap.String("mode", "production"))

	err = router.Run(":" + port)
	if err != nil {
		logger.Error("Failed to start server",
			zap.String("port", port),
			zap.Error(err))
	}
}
