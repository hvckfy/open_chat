package errofy

import (
	"account-service/services/logger"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LogError logs the error immediately when it occurs (for Loki)
// Use in internal functions to capture exact error location
func LogError(logicCode int64, err error, context string) {

	if Loki == false {
		fmt.Printf("Logic error occured with logic_code: %d context: %s error: %s", logicCode, context, err)
	} else {
		if logger.Log != nil {
			logger.Log.Error("Logic error occurred",
				zap.Int64("logic_code", logicCode),
				zap.String("context", context),
				zap.Error(err))
		}
	}
}

// RaiseApi returns the API error status and JSON response
func RaiseApiLogic(logicCode int64) (int, gin.H) {
	logicErr, exists := Errors[logicCode]
	if !exists {
		// Return default error
		return http.StatusInternalServerError, gin.H{"error": "Internal server error"}
	}

	return int(logicErr.ApiError.ApiErrorCode), gin.H{"error": logicErr.ApiError.ApiErrorDesc}
}

/*
// Internal example
error.LogError(4982, errors.Wrap(err, "database connection failed"), "GetUserFromDB")

// Handler example
error.LogError(1003, err, "LoginHandler") // log error
c.JSON(error.RaiseApi(1003)) // return to user
*/
