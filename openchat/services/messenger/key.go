package messenger

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"openchat/services/config"
	"openchat/services/entropy"
	"openchat/services/logger"
)

// return 2x string(both not encrypted) and error
func GenerateRSAStrings() (privStr string, pubStr string, err error) {
	// Generate private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	// Convert private key to string
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	}
	privStr = string(pem.EncodeToMemory(privBlock))

	// Convert public key to string
	pubBlock := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey),
	}
	pubStr = string(pem.EncodeToMemory(pubBlock))

	return privStr, pubStr, nil
}

// return str of encrypted message
func EncryptWithStr(pubStr string, input string) (encryptedMessage string, err error) {
	message := []byte(input)
	block, _ := pem.Decode([]byte(pubStr))
	pub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return "", err
	}
	// Encrypt using OAEP padding
	EncryptedBytes, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, message, nil)
	if err != nil {
		return "", err
	}
	encryptedMessage = string(EncryptedBytes)
	return encryptedMessage, nil
}

// decrypt message with RSA key as string
func DecryptWithStr(privStr string, input string) (decryptedMessage string, err error) {
	ciphertext := []byte(input)
	block, _ := pem.Decode([]byte(privStr))
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	// Decrypt using identical parameters
	decryptedBytes, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(decryptedBytes), nil
}

func GetKeys(user_id int64) (EncryptedRsaPrivKey string, RsaPubKey string, exists bool, err error) {
	logger.Info(fmt.Sprint(config.Data.Databases["MessageDb"], config.Data.Databases["MessageDb"].Port, config.Data.Databases["MessageDb"].User, config.Data.Databases["MessageDb"].Pass, config.Data.Databases["MessageDb"].Pass))
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Data.Databases["MessageDb"].Host,
		config.Data.Databases["MessageDb"].Port,
		config.Data.Databases["MessageDb"].User,
		config.Data.Databases["MessageDb"].Pass,
		config.Data.Databases["MessageDb"].Name)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return "", "", false, err
	}
	defer db.Close()

	err = db.QueryRow(
		"SELECT pub_key, priv_key FROM rsa_keys WHERE user_id = $1", user_id).
		Scan(&RsaPubKey, &EncryptedRsaPrivKey)
	if err == sql.ErrNoRows {
		return "", "", false, nil
	} else if err != nil {
		return "", "", false, err
	}
	return EncryptedRsaPrivKey, RsaPubKey, true, nil
}

/*
generated entropy_words if not exists
*/
func GenKeys(user_id int64) (words []string, PrivKey string, err error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Data.Databases["MessageDb"].Host,
		config.Data.Databases["MessageDb"].Port,
		config.Data.Databases["MessageDb"].User,
		config.Data.Databases["MessageDb"].Pass,
		config.Data.Databases["MessageDb"].Name)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return words, PrivKey, err
	}
	defer db.Close()
	privKey, pubKey, err := GenerateRSAStrings()
	if err != nil {
		return words, PrivKey, err
	}
	succes, entropy_, words, err := entropy.GeneratePair()
	if err != nil {
		return words, PrivKey, err
	}
	if !succes {
		return words, PrivKey, fmt.Errorf("Could not generate entropy pair")
	}
	PrivKey, err = entropy.EncryptString(privKey, entropy_)
	if err != nil {
		return words, PrivKey, err
	}

	//insert:
	_, err = db.Exec(
		"INSERT INTO rsa_keys (user_id, pub_key, priv_key) VALUES ($1, $2, $3)",
		user_id, pubKey, PrivKey)
	if err != nil {
		return words, PrivKey, err
	}
	//return user words and NOT encrypted private key
	return words, privKey, nil

}
