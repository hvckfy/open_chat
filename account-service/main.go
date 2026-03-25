package main

import (
	"account-service/handlers"
	"account-service/middleware"
	"account-service/services/config"
	"account-service/services/logger"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// Initialize config
	err := config.InitConfig()
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

	// Системные логи - простые функции без HTTP контекста
	logger.Info("Starting account service",
		zap.String("version", "1.0.0"),
		zap.String("port", "8080"))

	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auth routes
	router.POST("/login-ldap", handlers.LdapLogin)
	router.POST("/login-local", handlers.LocalLogin)
	router.POST("/register-local", handlers.LocalRegister)
	router.POST("/refresh-token", handlers.RefreshToken)
	router.DELETE("/revoke-token", handlers.RevokeToken)

	// Protected routes
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/profile", handlers.Profile)
	}

	port := "8080" // Default port
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
