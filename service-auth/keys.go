package serviceauth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// GenerateRSAKeyPair генерирует пару RSA ключей для сервиса
func GenerateRSAKeyPair(serviceName, keyDir string) error {
	// Создаем директорию если не существует
	if err := os.MkdirAll(keyDir, 0755); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Генерируем приватный ключ
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Сохраняем приватный ключ
	privateKeyPath := filepath.Join(keyDir, fmt.Sprintf("%s_private.pem", serviceName))
	if err := savePrivateKey(privateKey, privateKeyPath); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}

	// Сохраняем публичный ключ
	publicKeyPath := filepath.Join(keyDir, fmt.Sprintf("%s_public.pem", serviceName))
	if err := savePublicKey(&privateKey.PublicKey, publicKeyPath); err != nil {
		return fmt.Errorf("failed to save public key: %w", err)
	}

	fmt.Printf("✅ Generated RSA key pair for %s:\n", serviceName)
	fmt.Printf("   Private key: %s\n", privateKeyPath)
	fmt.Printf("   Public key:  %s\n", publicKeyPath)

	return nil
}

// savePrivateKey сохраняет приватный ключ в PEM формате
func savePrivateKey(privateKey *rsa.PrivateKey, filename string) error {
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, privateKeyPEM)
}

// savePublicKey сохраняет публичный ключ в PEM формате
func savePublicKey(publicKey *rsa.PublicKey, filename string) error {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}

	publicKeyPEM := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, publicKeyPEM)
}

// LoadPrivateKey загружает приватный ключ из файла
func LoadPrivateKey(filename string) (*rsa.PrivateKey, error) {
	keyData, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("invalid private key format")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return privateKey, nil
}

// LoadPublicKey загружает публичный ключ из файла
func LoadPublicKey(filename string) (*rsa.PublicKey, error) {
	keyData, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil || block.Type != "RSA PUBLIC KEY" {
		return nil, fmt.Errorf("invalid public key format")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPublicKey, nil
}

// LoadServicePublicKeys загружает публичные ключи всех сервисов
func LoadServicePublicKeys(keyDir string, services []string) (map[string]*rsa.PublicKey, error) {
	publicKeys := make(map[string]*rsa.PublicKey)

	for _, service := range services {
		keyPath := filepath.Join(keyDir, fmt.Sprintf("%s_public.pem", service))
		publicKey, err := LoadPublicKey(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load public key for %s: %w", service, err)
		}
		publicKeys[service] = publicKey
	}

	return publicKeys, nil
}
