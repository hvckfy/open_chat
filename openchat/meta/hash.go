package meta

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
)

// hash string to hashstring sha256
func HashString(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

/*
Gets value for environment if exists, otherwise return default value
*/
func GetEnvValue(key string, defaultValue string) string {
	envVal := os.Getenv(key)
	if envVal == "" {
		return defaultValue
	}
	return envVal
}
