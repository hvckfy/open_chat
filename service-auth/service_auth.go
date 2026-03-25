package serviceauth

import (
	"fmt"
	"log"
)

// ServiceAuth основная структура для работы с аутентификацией сервисов
type ServiceAuth struct {
	config ServiceConfig
	client *ServiceClient
	server *ServiceServer
}

// NewServiceAuth создает новый экземпляр ServiceAuth
func NewServiceAuth(config ServiceConfig) (*ServiceAuth, error) {
	client, err := NewServiceClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &ServiceAuth{
		config: config,
		client: client,
	}, nil
}

// NewServiceAuthServer создает сервер для обработки запросов
func NewServiceAuthServer(config ServiceConfig, services []string) (*ServiceAuth, error) {
	server, err := NewServiceServer(config, services)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	return &ServiceAuth{
		config: config,
		server: server,
	}, nil
}

// Client methods

// ValidateToken отправляет запрос на валидацию токена
func (sa *ServiceAuth) ValidateToken(userToken string, permissions []string, targetService string) (*ValidationResult, error) {
	return sa.client.ValidateToken(userToken, permissions, targetService)
}

// CheckPermissions проверяет права пользователя
func (sa *ServiceAuth) CheckPermissions(userID int, permissions []string, resource string, targetService string) (bool, error) {
	return sa.client.CheckPermissions(userID, permissions, resource, targetService)
}

// GetUser получает данные пользователя
func (sa *ServiceAuth) GetUser(userID int, targetService string) (map[string]interface{}, error) {
	return sa.client.GetUser(userID, targetService)
}

// SendRequest отправляет кастомный запрос
func (sa *ServiceAuth) SendRequest(action string, payload map[string]interface{}, targetService string) (*EncryptedMessage, error) {
	return sa.client.SendRequest(action, payload, targetService)
}

// Server methods

// Middleware возвращает Gin middleware для обработки запросов
func (sa *ServiceAuth) Middleware() interface{} {
	if sa.server == nil {
		log.Fatal("Server not initialized. Use NewServiceAuthServer()")
	}
	return sa.server.Middleware()
}

// HandleRequest обрабатывает запрос
func (sa *ServiceAuth) HandleRequest() interface{} {
	if sa.server == nil {
		log.Fatal("Server not initialized. Use NewServiceAuthServer()")
	}
	return sa.server.HandleRequest
}

// GetAPIKey возвращает API ключ для сервиса
func (sa *ServiceAuth) GetAPIKey(serviceName string) string {
	if sa.server == nil {
		log.Fatal("Server not initialized. Use NewServiceAuthServer()")
	}
	return sa.server.GetAPIKey(serviceName)
}

// Utility functions

// GenerateKeys генерирует RSA ключи для сервиса
func GenerateKeys(serviceName, keyDir string) error {
	return GenerateRSAKeyPair(serviceName, keyDir)
}

// SetupServiceKeys настраивает ключи для всех сервисов
func SetupServiceKeys(services []string, keyDir string) error {
	fmt.Printf("🔐 Setting up RSA keys for %d services...\n", len(services))

	for _, service := range services {
		if err := GenerateRSAKeyPair(service, keyDir); err != nil {
			return fmt.Errorf("failed to generate keys for %s: %w", service, err)
		}
	}

	fmt.Println("✅ All service keys generated successfully!")
	return nil
}

// ValidateSetup проверяет корректность настройки
func ValidateSetup(config ServiceConfig, services []string) error {
	// Проверяем приватный ключ
	if _, err := LoadPrivateKey(config.PrivateKeyPath); err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	// Проверяем публичный ключ
	if _, err := LoadPublicKey(config.PublicKeyPath); err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	// Проверяем ключи других сервисов
	for _, service := range services {
		keyPath := fmt.Sprintf("%s/%s_public.pem", config.KeyDir, service)
		if _, err := LoadPublicKey(keyPath); err != nil {
			return fmt.Errorf("invalid public key for %s: %w", service, err)
		}
	}

	fmt.Println("✅ Service authentication setup validated!")
	return nil
}
