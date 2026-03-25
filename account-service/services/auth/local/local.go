package local

import (
	"account-service/meta"
	"account-service/services/auth/user"
	"account-service/services/config"
	"fmt"
)

func VerifyUser(username, password string) (bool, int64, error) {
	u, exists, errorCode, err := user.GetUser(username)
	if err != nil {
		return false, errorCode, err
	}
	if !exists {
		return false, 4041, nil
	}
	// For local auth, check password hash
	if u.App.Password == meta.HashString(password) {
		return true, 200, nil
	}
	return false, 4011, nil
}

func ImportUser(username string) (bool, int64, error) {
	// For local auth, user should already be registered
	return true, 200, nil
}

/*
Auth user via local. Return refresh, access, error_code, error
*/
func AuthUser(username string, password string) (string, string, int64, error) {
	//check if users exists
	u, exists, error_code, err := user.GetUser(username)
	if err != nil {
		return "", "", error_code, err
	}
	if !exists {
		return "", "", 4041, fmt.Errorf("User doesnt exists")
	}
	if error_code != 200 {
		return "", "", error_code, err
	}
	if !(u.App.Password == meta.HashString(password)) {
		return "", "", 4011, fmt.Errorf("credentials doesnt match")
	}
	refreshToken, refreshExpireAt, error_code, err := user.GenerateJwt(username, config.Data.JWT.RefreshTokenExpire)
	if err != nil {
		return "", "", error_code, err
	}
	if error_code != 200 {
		return "", "", error_code, err
	}
	accessToken, _, error_code, err := user.GenerateJwt(username, config.Data.JWT.AccessTokenExpire)
	if err != nil {
		return "", "", 5005, err
	}
	if error_code != 200 {
		return "", "", error_code, err
	}
	success, error_code, err := user.AddRefreshJwt(u.App.UserId, refreshToken, refreshExpireAt)
	if err != nil {
		return "", "", error_code, err
	}
	if !success {
		return "", "", error_code, fmt.Errorf("unsuccess token registration")
	}
	if error_code != 200 {
		return "", "", error_code, err
	}
	return refreshToken, accessToken, 200, nil
}

/*
Register via local. Return refresh, access, error_code, error
*/
func RegisterUser(username string, password string) (string, string, int64, error) {
	_, exists, errorCode, err := user.GetUser(username)
	if err != nil {
		return "", "", errorCode, err
	}
	if exists {
		return "", "", 4091, fmt.Errorf("User already exists")
	}
	if errorCode != 200 {
		return "", "", errorCode, err
	}
	u := user.User{
		App: user.App{
			Username: username,
			Password: meta.HashString(password),
		},
	}
	u, errorCode, err = user.AddUser(u)
	if err != nil {
		return "", "", errorCode, err
	}
	if errorCode != 200 {
		return "", "", errorCode, err
	}
	refreshToken, refreshExpireAt, errorCode, err := user.GenerateJwt(username, config.Data.JWT.RefreshTokenExpire)
	if err != nil {
		return "", "", errorCode, err
	}
	if errorCode != 200 {
		return "", "", errorCode, err
	}
	accessToken, _, errorCode, err := user.GenerateJwt(username, config.Data.JWT.AccessTokenExpire)
	if err != nil {
		return "", "", 5005, err
	}
	if errorCode != 200 {
		return "", "", errorCode, err
	}
	success, errorCode, err := user.AddRefreshJwt(u.App.UserId, refreshToken, refreshExpireAt)
	if err != nil {
		return "", "", errorCode, err
	}
	if !success {
		return "", "", errorCode, fmt.Errorf("unsuccess token registration")
	}
	if errorCode != 200 {
		return "", "", errorCode, err
	}
	return refreshToken, accessToken, 200, nil
}
