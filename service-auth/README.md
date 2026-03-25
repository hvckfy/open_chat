# Service Auth Library

Библиотека для безопасной аутентификации между микросервисами с использованием RSA шифрования и API ключей.

## Особенности

- 🔐 **RSA шифрование** - запросы шифруются публичным ключом получателя
- 🛡️ **API ключи** - аутентификация сервисов
- ⚡ **Простой API** - легкая интеграция в существующие сервисы
- 🕒 **Защита от replay** - timestamp-based валидация
- 📝 **Гибкий формат** - поддержка кастомных запросов

## Архитектура

### Формат запроса:
```
POST /api/v1/service-request
Authorization: Bearer:{api_key}
message:{encrypted_payload}
```

### Формат ответа:
```json
{
  "response": {
    "data": "..."
  },
  "error": {
    "status": false,
    "error-code": "404",
    "message": "User not found"
  }
}
```

## Быстрый старт

### 1. Генерация ключей

```bash
# Создаем директорию для ключей
mkdir -p keys

# Генерируем ключи для сервисов
go run service-auth/main.go generate-keys order-service keys/
go run service-auth/main.go generate-keys user-service keys/
go run service-auth/main.go generate-keys account-service keys/
```

### 2. Настройка Account Service (сервера)

```go
package main

import (
    "github.com/your-org/service-auth"
    "github.com/gin-gonic/gin"
)

func main() {
    // Конфигурация
    config := serviceauth.ServiceConfig{
        ServiceName:    "account-service",
        APIKey:         "http://account-service:8080",
        PrivateKeyPath: "keys/account-service_private.pem",
        PublicKeyPath:  "keys/account-service_public.pem",
        KeyDir:         "keys",
    }

    services := []string{"order-service", "user-service"}

    // Создаем сервер
    auth, err := serviceauth.NewServiceAuthServer(config, services)
    if err != nil {
        panic(err)
    }

    // Настраиваем Gin
    r := gin.Default()

    // Middleware для обработки зашифрованных запросов
    r.Use(auth.Middleware().(gin.HandlerFunc))

    // Обработчик запросов
    r.POST("/api/v1/service-request", auth.HandleRequest().(gin.HandlerFunc))

    r.Run(":8080")
}
```

### 3. Настройка Order Service (клиента)

```go
package main

import (
    "github.com/your-org/service-auth"
)

func validateUserToken(userToken string) (*serviceauth.ValidationResult, error) {
    // Конфигурация
    config := serviceauth.ServiceConfig{
        ServiceName:    "order-service",
        APIKey:         "sk-order-1234567890abcdef",
        PrivateKeyPath: "keys/order-service_private.pem",
        PublicKeyPath:  "keys/order-service_public.pem",
        KeyDir:         "keys",
    }

    // Создаем клиент
    auth, err := serviceauth.NewServiceAuth(config)
    if err != nil {
        return nil, err
    }

    // Валидируем токен
    return auth.ValidateToken(userToken, []string{"read:orders"}, "account-service")
}

func main() {
    result, err := validateUserToken("jwt.token.here")
    if err != nil {
        panic(err)
    }

    if result.Valid {
        fmt.Printf("User %d validated\n", result.UserID)
    }
}
```

## API Reference

### ServiceAuth (Client)

```go
// Валидация токена
ValidateToken(userToken string, permissions []string, targetService string) (*ValidationResult, error)

// Проверка прав
CheckPermissions(userID int, permissions []string, resource string, targetService string) (bool, error)

// Получение пользователя
GetUser(userID int, targetService string) (map[string]interface{}, error)

// Кастомный запрос
SendRequest(action string, payload map[string]interface{}, targetService string) (*EncryptedMessage, error)
```

### ServiceAuth (Server)

```go
// Middleware для Gin
Middleware() gin.HandlerFunc

// Обработчик запросов
HandleRequest() gin.HandlerFunc

// Получение API ключа
GetAPIKey(serviceName string) string
```

## Примеры использования

### Валидация токена в Order Service

```go
func (h *OrderHandler) CreateOrder(c *gin.Context) {
    userToken := c.GetHeader("Authorization")

    // Валидируем токен через Account Service
    result, err := h.auth.ValidateToken(userToken, []string{"create:orders"}, "account-service")
    if err != nil || !result.Valid {
        c.JSON(401, gin.H{"error": "Invalid token"})
        return
    }

    // Создаем заказ для пользователя
    order := createOrder(result.UserID, c.Request.Body)
    c.JSON(200, order)
}
```

### Проверка прав доступа

```go
func (h *AdminHandler) DeleteUser(c *gin.Context) {
    userID := parseInt(c.Param("id"))

    // Проверяем права администратора
    hasAccess, err := h.auth.CheckPermissions(userID, []string{"admin:users"}, "user:"+c.Param("id"), "account-service")
    if err != nil || !hasAccess {
        c.JSON(403, gin.H{"error": "Access denied"})
        return
    }

    // Удаляем пользователя
    deleteUser(userID)
    c.JSON(200, gin.H{"status": "deleted"})
}
```

## Безопасность

- **Шифрование**: AES-256-GCM + RSA-OAEP
- **Аутентификация**: API ключи + цифровые подписи
- **Защита от replay**: timestamp validation (5 мин)
- **Конфиденциальность**: end-to-end шифрование

## Docker Compose пример

```yaml
version: '3.8'

services:
  account-service:
    build: ./account-service
    volumes:
      - ./keys:/app/keys:ro
    environment:
      - SERVICE_NAME=account-service
      - PRIVATE_KEY_PATH=/app/keys/account-service_private.pem
    ports:
      - "8080:8080"

  order-service:
    build: ./order-service
    volumes:
      - ./keys:/app/keys:ro
    environment:
      - SERVICE_NAME=order-service
      - PRIVATE_KEY_PATH=/app/keys/order-service_private.pem
    depends_on:
      - account-service
```

## Troubleshooting

### Ошибка "Failed to decrypt payload"
- Проверьте что публичный ключ сервиса доступен
- Убедитесь что ключи сгенерированы правильно
- Проверьте timestamp (не старше 5 минут)

### Ошибка "Invalid API key"
- Проверьте что API ключ настроен правильно
- Убедитесь что сервис зарегистрирован в списке services

### Ошибка "Unknown service"
- Добавьте сервис в список services при создании сервера
- Сгенерируйте ключи для нового сервиса

## Contributing

1. Fork the repository
2. Create feature branch
3. Add tests
4. Submit pull request

## License

MIT License