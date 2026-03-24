package user

import (
	"account-service/services/config"
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
Generate tokens
*/
func GenerateJwtToken(username string, durationSeconds int64) (string, error) {
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
		return "", err
	}
	return signedToken, nil
}

func 