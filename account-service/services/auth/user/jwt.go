package user

import (
	"account-service/meta"
	"account-service/services/config"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

/*
- GenerateAccessToken(username) — exp 15 мин

- GenerateRefreshToken(username) — exp 7 дней

- ValidateAccessToken(token) — проверка и возврат claims

- ValidateRefreshToken(token) — проверка и возврат claims
*/

/*
Generate tokens, returns signed_token, expire_at, error
->token, expire time, error
*/
func GenerateJwt(username string, durationSeconds int64) (string, int64, error) {
	//current time + duration in seconds
	expireTime := time.Now().Unix() + durationSeconds
	claims := jwt.MapClaims{
		"username": username,
		"exp":      expireTime,
	}
	//generate token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	//sign token by secret
	signedToken, err := token.SignedString([]byte(config.Data.JWT.Secret))
	if err != nil {
		return "", 0, err
	}
	return signedToken, expireTime, nil
}

/*
add refresh JWT tokens for user
->success, error
*/
func AddRefreshJwt(userId int64, refreshToken string, expireAt int64) (bool, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Data.DB.Host, config.Data.DB.Port, config.Data.DB.User, config.Data.DB.Pass, config.Data.DB.Name)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return false, err
	}
	defer db.Close()

	hashToken := meta.HashString(refreshToken)

	expireAtTime := time.Unix(expireAt, 0)

	query := `INSERT INTO refresh_tokens (user_id, token_hash, expires_at) 
              VALUES ($1, $2, $3) ON CONFLICT (token_hash) DO NOTHING`
	_, err = db.Exec(query, userId, hashToken, expireAtTime)
	return true, err
}

/*
validate access token for user with refresh token
->user, success, error
*/
func ValidateAccessJwt(accessToken string) (User, error) {
	// Parse and validate access token
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Data.JWT.Secret), nil
	})
	if err != nil {
		return User{}, err
	}
	if !token.Valid {
		return User{}, errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return User{}, errors.New("invalid claims")
	}
	username := claims["username"].(string)
	u, exists, err := GetUser(username)
	if err != nil {
		return User{}, err
	}
	if !exists {
		return User{}, errors.New("user not exists")
	}
	return u, nil
}

/*
validate refresh token for user
->access token, success, error
*/
func ValidateRefreshToken(refreshToken string) (string, error) {
	// Parse
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Data.JWT.Secret), nil
	})
	if err != nil || !token.Valid {
		return "", err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	username := claims["username"].(string)

	// Check in DB
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Data.DB.Host, config.Data.DB.Port, config.Data.DB.User, config.Data.DB.Pass, config.Data.DB.Name)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return "", err
	}
	defer db.Close()
	hash := meta.HashString(refreshToken)
	var expiresAt time.Time
	row := db.QueryRow("SELECT expires_at FROM refresh_tokens WHERE token_hash = $1", hash)
	err = row.Scan(&expiresAt)
	if err != nil {
		return "", err
	}
	if expiresAt.Unix() <= time.Now().Unix() {
		return "", errors.New("refresh expired")
	}

	// Generate new access
	access, _, err := GenerateJwt(username, config.Data.JWT.AccessTokenExpire)
	return access, err
}

/*
Terminate token
*/
func TerminateToken(refreshToken string) (bool, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Data.DB.Host, config.Data.DB.Port, config.Data.DB.User, config.Data.DB.Pass, config.Data.DB.Name)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return false, err
	}
	defer db.Close()

	hashToken := meta.HashString(refreshToken)

	query := `DELETE FROM refresh_tokens WHERE token_hash = $1`
	_, err = db.Exec(query, hashToken)
	return true, err
}
