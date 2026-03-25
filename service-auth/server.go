package serviceauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ServiceServer сервер для обработки зашифрованных запросов
type ServiceServer struct {
	config   ServiceConfig
	crypto   *AccountCrypto
	apiKeys  map[string]string // service_name -> api_key
	services []string
}

// NewServiceServer создает новый сервер для Account Service
func NewServiceServer(config ServiceConfig, services []string) (*ServiceServer, error) {
	crypto, err := NewAccountCrypto(config, services)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize crypto: %w", err)
	}

	// API ключи для сервисов (в проде - из конфига/базы)
	apiKeys := make(map[string]string)
	for _, service := range services {
		apiKeys[service] = fmt.Sprintf("sk-%s-1234567890abcdef", service)
	}

	return &ServiceServer{
		config:   config,
		crypto:   crypto,
		apiKeys:  apiKeys,
		services: services,
	}, nil
}

// Middleware возвращает Gin middleware для обработки service запросов
func (ss *ServiceServer) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer:") {
			c.JSON(http.StatusUnauthorized, EncryptedMessage{
				Error: &ErrorInfo{Status: true, ErrorCode: "401", Message: "Missing authorization"},
			})
			c.Abort()
			return
		}

		// Парсим: "Bearer:api_key\nmessage:encrypted_payload"
		parts := strings.Split(authHeader, "\n")
		if len(parts) != 2 || !strings.HasPrefix(parts[0], "Bearer:") || !strings.HasPrefix(parts[1], "message:") {
			c.JSON(http.StatusBadRequest, EncryptedMessage{
				Error: &ErrorInfo{Status: true, ErrorCode: "400", Message: "Invalid authorization format"},
			})
			c.Abort()
			return
		}

		apiKey := strings.TrimPrefix(parts[0], "Bearer:")
		encryptedPayload := strings.TrimPrefix(parts[1], "message:")

		// Находим сервис по API ключу
		serviceName, valid := ss.validateAPIKey(apiKey)
		if !valid {
			c.JSON(http.StatusUnauthorized, EncryptedMessage{
				Error: &ErrorInfo{Status: true, ErrorCode: "401", Message: "Invalid API key"},
			})
			c.Abort()
			return
		}

		// Расшифровываем payload
		decryptedData, err := ss.crypto.DecryptPayload(serviceName, encryptedPayload)
		if err != nil {
			c.JSON(http.StatusBadRequest, EncryptedMessage{
				Error: &ErrorInfo{Status: true, ErrorCode: "400", Message: "Failed to decrypt payload"},
			})
			c.Abort()
			return
		}

		// Парсим расшифрованный JSON
		var request ServiceRequest
		if err := json.Unmarshal(decryptedData, &request); err != nil {
			c.JSON(http.StatusBadRequest, EncryptedMessage{
				Error: &ErrorInfo{Status: true, ErrorCode: "400", Message: "Invalid payload format"},
			})
			c.Abort()
			return
		}

		// Проверяем timestamp (защита от replay attacks)
		if time.Now().Unix()-request.Timestamp > MaxRequestAge {
			c.JSON(http.StatusUnauthorized, EncryptedMessage{
				Error: &ErrorInfo{Status: true, ErrorCode: "401", Message: "Request expired"},
			})
			c.Abort()
			return
		}

		// Сохраняем данные в контексте для обработчика
		c.Set("service_name", serviceName)
		c.Set("service_request", request)
		c.Set("service_server", ss)

		c.Next()
	}
}

// HandleRequest обрабатывает расшифрованный запрос
func (ss *ServiceServer) HandleRequest(c *gin.Context) {
	serviceName := c.MustGet("service_name").(string)
	request := c.MustGet("service_request").(ServiceRequest)

	// Логируем запрос
	fmt.Printf("Service request: %s from %s (%s)\n",
		request.Action, serviceName, request.RequestID)

	var response EncryptedMessage

	switch request.Action {
	case ActionValidateToken:
		response = ss.handleValidateToken(request.Payload)
	case ActionGetUser:
		response = ss.handleGetUser(request.Payload)
	case ActionCheckPermissions:
		response = ss.handleCheckPermissions(request.Payload)
	case ActionRefreshToken:
		response = ss.handleRefreshToken(request.Payload)
	default:
		response = EncryptedMessage{
			Error: &ErrorInfo{Status: true, ErrorCode: "400", Message: "Unknown action: " + request.Action},
		}
	}

	c.JSON(http.StatusOK, response)
}

// handleValidateToken валидирует пользовательский JWT токен
func (ss *ServiceServer) handleValidateToken(payload map[string]interface{}) EncryptedMessage {
	_, ok := payload["user_token"].(string)
	if !ok {
		return EncryptedMessage{
			Error: &ErrorInfo{Status: true, ErrorCode: "400", Message: "Missing user_token"},
		}
	}

	// Здесь должна быть логика валидации токена
	// Для примера - просто возвращаем успех
	return EncryptedMessage{
		Response: map[string]interface{}{
			"valid":        true,
			"user_id":      123,
			"username":     "testuser",
			"validated_by": "service-auth",
		},
	}
}

// handleGetUser получает данные пользователя
func (ss *ServiceServer) handleGetUser(payload map[string]interface{}) EncryptedMessage {
	userID, ok := payload["user_id"].(float64)
	if !ok {
		return EncryptedMessage{
			Error: &ErrorInfo{Status: true, ErrorCode: "400", Message: "Missing user_id"},
		}
	}

	// Здесь должна быть логика получения пользователя из базы
	// Для примера - возвращаем mock данные
	return EncryptedMessage{
		Response: map[string]interface{}{
			"user": map[string]interface{}{
				"id":       int(userID),
				"username": "testuser",
				"email":    "test@example.com",
				"role":     "customer",
			},
		},
	}
}

// handleCheckPermissions проверяет права пользователя
func (ss *ServiceServer) handleCheckPermissions(payload map[string]interface{}) EncryptedMessage {
	userID, ok := payload["user_id"].(float64)
	if !ok {
		return EncryptedMessage{
			Error: &ErrorInfo{Status: true, ErrorCode: "400", Message: "Missing user_id"},
		}
	}

	permissions, ok := payload["permissions"].([]interface{})
	if !ok {
		return EncryptedMessage{
			Error: &ErrorInfo{Status: true, ErrorCode: "400", Message: "Missing permissions"},
		}
	}

	resource, _ := payload["resource"].(string)

	// Здесь должна быть логика проверки прав
	// Для примера - возвращаем успех
	return EncryptedMessage{
		Response: map[string]interface{}{
			"has_access":  true,
			"user_id":     int(userID),
			"permissions": permissions,
			"resource":    resource,
		},
	}
}

// handleRefreshToken обновляет токен
func (ss *ServiceServer) handleRefreshToken(payload map[string]interface{}) EncryptedMessage {
	refreshToken, ok := payload["refresh_token"].(string)
	if !ok {
		return EncryptedMessage{
			Error: &ErrorInfo{Status: true, ErrorCode: "400", Message: "Missing refresh_token"},
		}
	}

	// Здесь должна быть логика обновления токена
	// Для примера - возвращаем новый токен
	return EncryptedMessage{
		Response: map[string]interface{}{
			"access_token":  "new.jwt.token",
			"refresh_token": refreshToken, // обычно новый
			"expires_in":    3600,
		},
	}
}

// validateAPIKey проверяет API ключ и возвращает имя сервиса
func (ss *ServiceServer) validateAPIKey(apiKey string) (string, bool) {
	for service, key := range ss.apiKeys {
		if key == apiKey {
			return service, true
		}
	}
	return "", false
}

// GetAPIKey возвращает API ключ для сервиса (для тестирования)
func (ss *ServiceServer) GetAPIKey(serviceName string) string {
	return ss.apiKeys[serviceName]
}
