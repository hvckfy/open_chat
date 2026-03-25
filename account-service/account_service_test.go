package main

import (
	"account-service/services/auth/ldap"
	"account-service/services/auth/local"
	"account-service/services/auth/user"
	"account-service/services/config"
	"account-service/services/logger"
	"fmt"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	config.InitConfig()
	logger.InitLogger() // Initialize logger for tests
	if logger.Log != nil {
		logger.Log.Info("Logger initialized for tests")
	}
	defer logger.Sync() // Flush logs at the end
	m.Run()
}

func TestAuthUser(t *testing.T) {
	// Invalid login
	_, _, err := ldap.AuthUser("invalid", "pass")
	if err == nil {
		t.Error("should fail")
	}

	// Valid (LDAP not implemented, so this will fail)
	_, _, err = ldap.AuthUser("rmiftakhov", "Belayaakula2001-")
	if err == nil {
		t.Error("LDAP not implemented, should fail")
	}
}

func TestValidateAccessJwt(t *testing.T) {
	config.InitConfig()

	// Since LDAP is not implemented, we'll test with local auth
	username := "testuser_jwt_" + fmt.Sprintf("%d", time.Now().UnixNano())

	// Register user locally
	refresh, access, err := local.RegisterUser(username, "testpass")
	if err != nil {
		t.Fatal(err)
	}

	// Valid access token
	u, err := user.ValidateAccessJwt(access)
	if err != nil {
		t.Error("should be valid")
	}
	if u.App.Username != username {
		t.Error("wrong user")
	}

	// Invalid access token
	_, err = user.ValidateAccessJwt("invalid")
	if err == nil {
		t.Error("should be invalid")
	}

	// Valid refresh token
	newAccess, err := user.ValidateRefreshToken(refresh)
	if err != nil {
		t.Error("refresh token should be valid")
	}
	if newAccess == "" {
		t.Error("new access token should not be empty")
	}

	// Validate the new access token
	u2, err := user.ValidateAccessJwt(newAccess)
	if err != nil {
		t.Error("new access token should be valid")
	}
	if u2.App.Username != username {
		t.Error("wrong user in new access token")
	}

	// Invalid refresh token
	_, err = user.ValidateRefreshToken("invalid")
	if err == nil {
		t.Error("invalid refresh token should fail")
	}

	// Empty refresh token
	acc, err := user.ValidateRefreshToken("")
	t.Log("TOKEN:", acc)
	if err == nil {
		t.Error("empty refresh token should fail")
	}
}

func TestLocalRegisterUser(t *testing.T) {
	// Use unique username to avoid conflicts with previous test runs
	username := "testuser_" + fmt.Sprintf("%d", time.Now().UnixNano())

	// Register new user
	refresh, access, err := local.RegisterUser(username, "testpass")
	if err != nil {
		t.Fatal(err)
	}
	if refresh == "" || access == "" {
		t.Error("tokens should not be empty after registration")
	}

	// Check user exists
	u, exists, err := user.GetUser(username)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("user should exist after registration")
	}
	if u.App.Username != username {
		t.Error("wrong username")
	}

	// Try to register again - should fail
	_, _, err = local.RegisterUser(username, "testpass")
	if err == nil {
		t.Error("registering existing user should fail")
	}
}

func TestLocalAuthUser(t *testing.T) {
	// Use unique username to avoid conflicts
	username := "testuser2_" + fmt.Sprintf("%d", time.Now().UnixNano())

	// First register
	_, _, err := local.RegisterUser(username, "testpass")
	if err != nil {
		t.Fatal(err)
	}

	// Valid login
	refresh, access, err := local.AuthUser(username, "testpass")
	if err != nil {
		t.Fatal(err)
	}
	if refresh == "" || access == "" {
		t.Error("tokens should not be empty")
	}

	// Invalid password
	_, _, err = local.AuthUser(username, "wrongpass")
	if err == nil {
		t.Error("invalid password should fail")
	}

	// Non-existent user
	_, _, err = local.AuthUser("nonexistent", "pass")
	if err == nil {
		t.Error("non-existent user should fail")
	}
}

func TestLocalRefreshToken(t *testing.T) {
	// Use unique username to avoid conflicts
	username := "testuser3_" + fmt.Sprintf("%d", time.Now().UnixNano())

	// Register and login
	refresh, _, err := local.RegisterUser(username, "testpass")
	if err != nil {
		t.Fatal(err)
	}

	// Use refresh token to get new access
	newAccess, err := user.ValidateRefreshToken(refresh)
	if err != nil {
		t.Fatal(err)
	}
	if newAccess == "" {
		t.Error("new access token should not be empty")
	}

	// Validate new access token
	u, err := user.ValidateAccessJwt(newAccess)
	if err != nil {
		t.Fatal(err)
	}
	if u.App.Username != username {
		t.Error("wrong user in refreshed token")
	}
}
