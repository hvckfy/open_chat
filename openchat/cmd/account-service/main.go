package main

import (
	"log"
	"net/http"
	"openchat/handlers"
	"openchat/middleware"
	"openchat/services/config"
	"openchat/services/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// Initialize config
	err := config.InitAccountServiceConfig()
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

	logger.Info("Starting account service",
		zap.String("version", "1.0.0"),
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

	// Test UI
	router.StaticFile("/test-ui", "test-ui.html")

	// Auth routes
	router.POST("/login-ldap", handlers.LdapLogin)
	router.POST("/login-local", handlers.LocalLogin)
	router.POST("/register-local", handlers.LocalRegister)

	// Public refresh/revoke (JSON body)
	router.POST("/refresh-token", handlers.RefreshToken)
	router.DELETE("/revoke-token", handlers.RevokeToken)
	router.DELETE("/revoke-all-tokens", handlers.RevokeAllTokens)

	// Cookie protected API
	api := router.Group("/api")
	api.Use(middleware.CookieAuthMiddleware())
	api.GET("/profile", handlers.Profile)

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
