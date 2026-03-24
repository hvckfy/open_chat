package meta

import (
	"crypto/sha256"
	"encoding/hex"
)

// hash string to hashstring sha256
func HashString(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
