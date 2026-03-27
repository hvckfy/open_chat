package main

import (
	"fmt"
	"log"
	"net/http"

	"openchat/handlers"
	middleware "openchat/middleware/account-service"
	"openchat/services/config"
	"openchat/services/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {

	err := config.InitAccountServiceConfig()
	if err != nil {
		log.Printf("ERROR: Failed to initialize config: %v", err)
		return
	}

	err = logger.InitLogger()
	if err != nil {
		log.Printf("ERROR: Failed to initialize logger: %v", err)
		return
	}
	defer logger.Sync()

	router := gin.Default()

	// CORS
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

	// health
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auth routes
	apiAuth := router.Group("/auth")
	apiAuth.POST("/login-ldap", handlers.LdapLogin)
	apiAuth.POST("/login-local", handlers.LocalLogin)
	apiAuth.POST("/register-local", handlers.LocalRegister)

	// Public refresh/revoke
	apiPublic := router.Group("/public")
	apiPublic.POST("/refresh-token", handlers.RefreshToken)
	apiPublic.DELETE("/revoke-token", handlers.RevokeToken)
	apiPublic.DELETE("/revoke-all-tokens", handlers.RevokeAllTokens)

	// Cookie protected
	apiProtected := router.Group("/protected")
	apiProtected.Use(middleware.CookieAuthMiddleware())
	apiProtected.GET("/profile", handlers.Profile)

	go func() {
		port := config.Data.Service.Port
		logger.Info("HTTP server starting",
			zap.String("port", port))
		err := http.ListenAndServe(":"+port, router)
		if err != nil {
			logger.Error("HTTP server failed", zap.Error(err))
		}
	}()

	mtlsRouter := gin.Default()

	// INTERNAL API (for service-service comms)
	internal := mtlsRouter.Group("/api/account/service")
	internal.POST("/verify-access-token", handlers.ServiceAuth())

	mtlsServer := &http.Server{
		Addr:      fmt.Sprintf(":%s", config.Data.MtlsPort),
		Handler:   mtlsRouter,
		TLSConfig: config.Data.Mtls,
	}

	logger.Info("mTLS server starting",
		zap.String("port", config.Data.MtlsPort))

	err = mtlsServer.ListenAndServeTLS("", "")
	if err != nil {
		logger.Error("mTLS server failed", zap.Error(err))
	}
}
