package serviceauth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// ServiceCrypto обрабатывает шифрование для конкретного сервиса
type ServiceCrypto struct {
	privateKey  *rsa.PrivateKey
	serviceName string
}

// NewServiceCrypto создает новый экземпляр для шифрования
func NewServiceCrypto(config ServiceConfig) (*ServiceCrypto, error) {
	privateKey, err := LoadPrivateKey(config.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	return &ServiceCrypto{
		privateKey:  privateKey,
		serviceName: config.ServiceName,
	}, nil
}

// EncryptPayload шифрует JSON payload с помощью публичного ключа Account Service
func (sc *ServiceCrypto) EncryptPayload(payload []byte, accountPublicKey *rsa.PublicKey) (string, error) {
	// Генерируем случайный AES ключ (32 байта для AES-256)
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return "", fmt.Errorf("failed to generate AES key: %w", err)
	}

	// Шифруем payload с помощью AES
	encryptedPayload, err := encryptAES(payload, aesKey)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt payload with AES: %w", err)
	}

	// Шифруем AES ключ с помощью публичного ключа Account Service
	encryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, accountPublicKey, aesKey, nil)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt AES key with RSA: %w", err)
	}

	// Комбинируем: encrypted_key + encrypted_payload
	combined := append(encryptedKey, encryptedPayload...)

	// Кодируем в base64 для безопасной передачи
	return base64.StdEncoding.EncodeToString(combined), nil
}

// AccountCrypto обрабатывает расшифровку для Account Service
type AccountCrypto struct {
	privateKey *rsa.PrivateKey           // Приватный ключ Account Service
	publicKeys map[string]*rsa.PublicKey // Публичные ключи сервисов для проверки
}

// NewAccountCrypto создает новый экземпляр для расшифровки
func NewAccountCrypto(config ServiceConfig, services []string) (*AccountCrypto, error) {
	// Загружаем приватный ключ Account Service
	privateKey, err := LoadPrivateKey(config.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load account private key: %w", err)
	}

	// Загружаем публичные ключи сервисов
	publicKeys, err := LoadServicePublicKeys(config.KeyDir, services)
	if err != nil {
		return nil, fmt.Errorf("failed to load service public keys: %w", err)
	}

	return &AccountCrypto{
		privateKey: privateKey,
		publicKeys: publicKeys,
	}, nil
}

// DecryptPayload расшифровывает payload от конкретного сервиса
func (ac *AccountCrypto) DecryptPayload(serviceName, encryptedData string) ([]byte, error) {
	// Декодируем из base64
	combined, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted data: %w", err)
	}

	// Проверяем что сервис известен
	if _, exists := ac.publicKeys[serviceName]; !exists {
		return nil, fmt.Errorf("unknown service: %s", serviceName)
	}

	// Разделяем encrypted_key и encrypted_payload
	keySize := ac.privateKey.PublicKey.Size()
	if len(combined) < keySize {
		return nil, fmt.Errorf("encrypted data too short")
	}

	encryptedKey := combined[:keySize]
	encryptedPayload := combined[keySize:]

	// Расшифровываем AES ключ приватным ключом Account Service
	aesKey, err := rsa.DecryptOAEP(sha256.New(), nil, ac.privateKey, encryptedKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt AES key: %w", err)
	}

	// Расшифровываем payload с помощью AES
	plaintext, err := decryptAES(encryptedPayload, aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt payload: %w", err)
	}

	return plaintext, nil
}

// encryptAES шифрует данные с помощью AES-GCM
func encryptAES(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptAES расшифровывает данные с помощью AES-GCM
func decryptAES(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
