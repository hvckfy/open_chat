package user

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"openchat/meta"
	"openchat/services/config"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq" // PostgreSQL driver
)

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
		log.Printf("ERROR GenerateJwt: failed to sign token for %s: %v", username, err)
		return "", 0, fmt.Errorf("failed to generate token: %w", err)
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
		log.Printf("ERROR AddRefreshJwt: database connection failed: %v", err)
		return false, fmt.Errorf("database connection failed: %w", err)
	}
	defer db.Close()

	hashToken := meta.HashString(refreshToken)

	expireAtTime := time.Unix(expireAt, 0)

	query := `INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
              VALUES ($1, $2, $3) ON CONFLICT (token_hash) DO NOTHING`
	_, err = db.Exec(query, userId, hashToken, expireAtTime)
	if err != nil {
		log.Printf("ERROR AddRefreshJwt: failed to insert refresh token: %v", err)
		return false, fmt.Errorf("failed to store refresh token: %w", err)
	}
	return true, nil
}

/*
validate access token for user
->user, error
*/
func ValidateAccessJwt(accessToken string) (User, error) {
	// Parse and validate access token
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Data.JWT.Secret), nil
	})
	if err != nil {
		log.Printf("ERROR ValidateAccessJwt: invalid token: %v", err)
		return User{}, fmt.Errorf("invalid access token: %w", err)
	}
	if !token.Valid {
		return User{}, fmt.Errorf("token is not valid")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return User{}, fmt.Errorf("invalid token claims")
	}
	username := claims["username"].(string)
	u, exists, err := GetUser(username)
	if err != nil {
		return User{}, fmt.Errorf("failed to get user: %w", err)
	}
	if !exists {
		return User{}, fmt.Errorf("user not found")
	}
	return u, nil
}

/*
validate refresh token for user
->access token, success, error
*/
func ValidateRefreshToken(refreshToken string) (string, error) {
	username, err := ParseUsername(refreshToken)
	if err != nil {
		return "", err
	}
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
	if err != nil {
		return "", err
	}
	return access, nil
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

	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Data.JWT.Secret), nil
	})
	if err != nil || !token.Valid {
		return false, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false, errors.New("invalid claims")
	}
	username := claims["username"].(string)

	tokens, err := GetTokens(username)
	if err != nil {
		return false, err
	}
	hashToken := meta.HashString(refreshToken)
	if !slices.Contains(tokens, hashToken) {
		return false, errors.New("User with this token doesn't exist")
	}

	query := `DELETE FROM refresh_tokens WHERE token_hash = $1`
	_, err = db.Exec(query, hashToken)
	if err != nil {
		return false, err
	}
	return true, nil
}

func GetTokens(username string) ([]string, error) {
	var tokens []string
	u, exists, err := GetUser(username)
	if err != nil {
		return tokens, err
	}
	if !exists {
		return tokens, errors.New("user not found")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Data.DB.Host, config.Data.DB.Port, config.Data.DB.User, config.Data.DB.Pass, config.Data.DB.Name)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return tokens, fmt.Errorf("database connection failed: %w", err)
	}
	defer db.Close()

	query := `SELECT token_hash FROM refresh_tokens WHERE user_id = $1`
	rows, err := db.Query(query, u.App.UserId)
	if err != nil {
		return tokens, fmt.Errorf("database query failed for user %s: %w", username, err)
	}
	defer rows.Close()

	for rows.Next() {
		var tokenHash string
		if err := rows.Scan(&tokenHash); err != nil {
			return tokens, fmt.Errorf("failed to scan token hash: %w", err)
		}
		tokens = append(tokens, tokenHash)
	}
	if err := rows.Err(); err != nil {
		return tokens, fmt.Errorf("rows iteration error: %w", err)
	}
	return tokens, nil
}

func TerminateAll(tokenNotToTerminate string) (bool, error) {
	username, err := ParseUsername(tokenNotToTerminate)
	if err != nil {
		return false, err
	}
	u, ok, err := GetUser(username)
	if err != nil {
		return false, nil
	}
	if !ok {
		return ok, fmt.Errorf("Not ok getting user")
	}

	Tokens, err := GetTokens(username)
	if err != nil {
		return false, err
	}

	hashToken := meta.HashString(tokenNotToTerminate)

	if !slices.Contains(Tokens, hashToken) {
		fmt.Println("NOT CONTAINTS")
		return false, nil
	}
	fmt.Println(Tokens, hashToken)

	query := `DELETE FROM refresh_tokens WHERE token_hash != $1 AND user_id = $2`
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Data.DB.Host, config.Data.DB.Port, config.Data.DB.User, config.Data.DB.Pass, config.Data.DB.Name)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return false, err
	}
	defer db.Close()
	_, err = db.Exec(query, meta.HashString(hashToken), u.App.UserId)
	if err != nil {
		return false, err
	}
	return true, nil

}

func ParseUsername(refreshToken string) (string, error) {
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
	return username, nil
}
