package serviceauth

// ServiceRequest представляет запрос между сервисами
type ServiceRequest struct {
	Action    string                 `json:"action"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp int64                  `json:"timestamp"`
	ServiceID string                 `json:"service_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
}

// ServiceResponse представляет ответ от сервиса
type ServiceResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// ServiceConfig конфигурация для сервиса
type ServiceConfig struct {
	ServiceName    string
	APIKey         string
	PrivateKeyPath string
	PublicKeyPath  string
	KeyDir         string
}

// EncryptedMessage зашифрованное сообщение между сервисами
type EncryptedMessage struct {
	Response map[string]interface{} `json:"response,omitempty"`
	Error    *ErrorInfo             `json:"error,omitempty"`
}

// ErrorInfo информация об ошибке
type ErrorInfo struct {
	Status    bool   `json:"status"`
	ErrorCode string `json:"error-code,omitempty"`
	Message   string `json:"message,omitempty"`
}

// ValidationResult результат валидации токена
type ValidationResult struct {
	Valid  bool   `json:"valid"`
	UserID int    `json:"user_id,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Permissions проверка прав доступа
type Permissions struct {
	UserID      int      `json:"user_id"`
	Permissions []string `json:"permissions"`
	Resource    string   `json:"resource,omitempty"`
}

// Constants
const (
	ActionValidateToken    = "validate_token"
	ActionGetUser          = "get_user"
	ActionCheckPermissions = "check_permissions"
	ActionRefreshToken     = "refresh_token"

	MaxRequestAge = 300 // 5 minutes in seconds
)
