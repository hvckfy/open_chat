package local

import (
	"fmt"
	"log"
	"openchat/meta"
	"openchat/services/auth/user"
	"openchat/services/config"
)

func VerifyUser(username, password string) (bool, error) {
	u, exists, err := user.GetUser(username)
	if err != nil {
		log.Printf("ERROR VerifyUser: failed to get user %s: %v", username, err)
		return false, fmt.Errorf("failed to verify user: %w", err)
	}
	if !exists {
		return false, nil // User not found - not an error for verification
	}
	// For local auth, check password hash
	if u.App.Password == meta.HashString(password) {
		return true, nil
	}
	return false, nil // Wrong password - not an error, just invalid credentials
}

func ImportUser(username string) (bool, error) {
	// For local auth, user should already be registered
	return true, nil
}

/*
Auth user via local. Return refresh, access, error
*/
func AuthUser(username string, password string) (string, string, error) {
	//check if users exists
	u, exists, err := user.GetUser(username)
	if err != nil {
		log.Printf("ERROR AuthUser: failed to get user %s: %v", username, err)
		return "", "", fmt.Errorf("authentication failed: %w", err)
	}
	if !exists {
		return "", "", fmt.Errorf("user not found")
	}
	if u.App.AuthType != "local" {
		return "", "", fmt.Errorf("Not allowed auth type")
	}
	if !(u.App.Password == meta.HashString(password)) {
		return "", "", fmt.Errorf("invalid credentials")
	}
	refreshToken, refreshExpireAt, err := user.GenerateJwt(username, config.Data.JWT.RefreshTokenExpire)
	if err != nil {
		log.Printf("ERROR AuthUser: failed to generate refresh token for %s: %v", username, err)
		return "", "", fmt.Errorf("failed to generate tokens: %w", err)
	}
	accessToken, _, err := user.GenerateJwt(username, config.Data.JWT.AccessTokenExpire)
	if err != nil {
		log.Printf("ERROR AuthUser: failed to generate access token for %s: %v", username, err)
		return "", "", fmt.Errorf("failed to generate tokens: %w", err)
	}
	success, err := user.AddRefreshJwt(u.App.UserId, refreshToken, refreshExpireAt)
	if err != nil {
		log.Printf("ERROR AuthUser: failed to store refresh token for %s: %v", username, err)
		return "", "", fmt.Errorf("failed to store tokens: %w", err)
	}
	if !success {
		return "", "", fmt.Errorf("failed to register refresh token")
	}
	return refreshToken, accessToken, nil
}

/*
Register via local. Return refresh, access, error
*/
func RegisterUser(username string, password string) (string, string, error) {
	_, exists, err := user.GetUser(username)
	if err != nil {
		log.Printf("ERROR RegisterUser: failed to check user %s: %v", username, err)
		return "", "", fmt.Errorf("registration check failed: %w", err)
	}
	if exists {
		return "", "", fmt.Errorf("user already exists")
	}
	u := user.User{
		App: user.App{
			Username: username,
			Password: meta.HashString(password),
			AuthType: "local",
		},
	}
	u, err = user.AddUser(u)
	if err != nil {
		log.Printf("ERROR RegisterUser: failed to create user %s: %v", username, err)
		return "", "", fmt.Errorf("failed to create user: %w", err)
	}
	refreshToken, refreshExpireAt, err := user.GenerateJwt(username, config.Data.JWT.RefreshTokenExpire)
	if err != nil {
		log.Printf("ERROR RegisterUser: failed to generate refresh token for %s: %v", username, err)
		return "", "", fmt.Errorf("failed to generate tokens: %w", err)
	}
	accessToken, _, err := user.GenerateJwt(username, config.Data.JWT.AccessTokenExpire)
	if err != nil {
		log.Printf("ERROR RegisterUser: failed to generate access token for %s: %v", username, err)
		return "", "", fmt.Errorf("failed to generate tokens: %w", err)
	}
	success, err := user.AddRefreshJwt(u.App.UserId, refreshToken, refreshExpireAt)
	if err != nil {
		log.Printf("ERROR RegisterUser: failed to store refresh token for %s: %v", username, err)
		return "", "", fmt.Errorf("failed to store tokens: %w", err)
	}
	if !success {
		return "", "", fmt.Errorf("failed to register refresh token")
	}
	return refreshToken, accessToken, nil
}
