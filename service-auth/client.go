package serviceauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ServiceClient клиент для отправки зашифрованных запросов
type ServiceClient struct {
	config     ServiceConfig
	crypto     *ServiceCrypto
	httpClient *http.Client
}

// NewServiceClient создает новый клиент для сервиса
func NewServiceClient(config ServiceConfig) (*ServiceClient, error) {
	crypto, err := NewServiceCrypto(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize crypto: %w", err)
	}

	return &ServiceClient{
		config: config,
		crypto: crypto,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// SendRequest отправляет зашифрованный запрос к другому сервису
func (sc *ServiceClient) SendRequest(action string, payload map[string]interface{}, targetService string) (*EncryptedMessage, error) {
	// Создаем запрос
	request := ServiceRequest{
		Action:    action,
		Payload:   payload,
		Timestamp: time.Now().Unix(),
		ServiceID: sc.config.ServiceName,
		RequestID: generateRequestID(),
	}

	// Сериализуем в JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Загружаем публичный ключ целевого сервиса
	targetPublicKeyPath := fmt.Sprintf("%s/%s_public.pem", sc.config.KeyDir, targetService)
	targetPublicKey, err := LoadPublicKey(targetPublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load target service public key: %w", err)
	}

	// Шифруем payload публичным ключом целевого сервиса
	encryptedPayload, err := sc.crypto.EncryptPayload(jsonData, targetPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt payload: %w", err)
	}

	// Создаем HTTP запрос с кастомным Authorization header
	req, err := http.NewRequest("POST", sc.config.APIKey+"/api/v1/service-request", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Устанавливаем кастомный Authorization header
	authHeader := fmt.Sprintf("Bearer:%s\nmessage:%s", sc.config.APIKey, encryptedPayload)
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("X-Source-Service", sc.config.ServiceName)
	req.Header.Set("X-Target-Service", targetService)

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Читаем тело ответа (незашифрованный JSON)
	var encryptedMessage EncryptedMessage
	if err := json.NewDecoder(resp.Body).Decode(&encryptedMessage); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &encryptedMessage, nil
}

// ValidateToken отправляет запрос на валидацию токена
func (sc *ServiceClient) ValidateToken(userToken string, permissions []string, targetService string) (*ValidationResult, error) {
	payload := map[string]interface{}{
		"user_token":  userToken,
		"permissions": permissions,
	}

	response, err := sc.SendRequest(ActionValidateToken, payload, targetService)
	if err != nil {
		return nil, err
	}

	// Проверяем на ошибки
	if response.Error != nil && response.Error.Status {
		return &ValidationResult{Error: response.Error.Message}, nil
	}

	// Парсим результат валидации
	result := &ValidationResult{}
	if response.Response != nil {
		if userID, ok := response.Response["user_id"].(float64); ok {
			result.UserID = int(userID)
		}
		if valid, ok := response.Response["valid"].(bool); ok {
			result.Valid = valid
		}
	}

	return result, nil
}

// CheckPermissions проверяет права пользователя
func (sc *ServiceClient) CheckPermissions(userID int, permissions []string, resource string, targetService string) (bool, error) {
	payload := map[string]interface{}{
		"user_id":     userID,
		"permissions": permissions,
		"resource":    resource,
	}

	response, err := sc.SendRequest(ActionCheckPermissions, payload, targetService)
	if err != nil {
		return false, err
	}

	// Проверяем на ошибки
	if response.Error != nil && response.Error.Status {
		return false, fmt.Errorf("permission check failed: %s", response.Error.Message)
	}

	if response.Response != nil {
		if hasAccess, ok := response.Response["has_access"].(bool); ok {
			return hasAccess, nil
		}
	}

	return false, fmt.Errorf("invalid response format")
}

// GetUser получает данные пользователя
func (sc *ServiceClient) GetUser(userID int, targetService string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"user_id": userID,
	}

	response, err := sc.SendRequest(ActionGetUser, payload, targetService)
	if err != nil {
		return nil, err
	}

	// Проверяем на ошибки
	if response.Error != nil && response.Error.Status {
		return nil, fmt.Errorf("get user failed: %s", response.Error.Message)
	}

	if response.Response != nil {
		if userData, ok := response.Response["user"].(map[string]interface{}); ok {
			return userData, nil
		}
	}

	return nil, fmt.Errorf("invalid response format")
}

// generateRequestID генерирует уникальный ID запроса
func generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}
