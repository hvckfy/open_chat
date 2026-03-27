package entropy

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/tyler-smith/go-bip39"
)

func GeneratePair() (success bool, entropy []byte, words []string, err error) {
	// Generate 256 bits of entropy
	entropy, err = bip39.NewEntropy(256)
	if err != nil {
		return false, nil, nil, err
	}
	words, err = EntropyToWords(entropy)
	if err != nil {
		return false, entropy, nil, err
	}
	return true, entropy, words, nil
}

// usage is transfer pass-wprds from entropy
func EntropyToWords(entropy []byte) (words []string, err error) {
	// Convert entropy to mnemonic
	phrase, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return nil, err
	}

	words = strings.Fields(phrase)
	return words, nil
}

// usage is transfer pass-words to entropy
func WordsToEntropy(words []string) ([]byte, error) {
	phrase := strings.Join(words, " ")
	return bip39.MnemonicToByteArray(phrase)
}

// usage is encrypting RSA key
func EncryptString(s string, rawKey []byte) (string, error) {
	h := sha256.Sum256(rawKey[:32])
	key := h[:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(s), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// usage is decryptin RSA key
func DecryptString(ciphertextStr string, rawKey []byte) (string, error) {
	h := sha256.Sum256(rawKey[:32])
	key := h[:]

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextStr)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	ct := ciphertext[nonceSize:]

	plaintext, err := aesGCM.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
