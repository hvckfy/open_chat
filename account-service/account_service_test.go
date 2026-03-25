package main

import (
	"account-service/services/auth/ldap"
	"account-service/services/auth/local"
	"account-service/services/auth/user"
	"account-service/services/config"
	"testing"
)

func TestMain(m *testing.M) {
	config.InitConfig()
	m.Run()
}

func TestAuthUser(t *testing.T) {
	// Invalid login
	_, _, _, err := ldap.AuthUser("invalid", "pass")
	if err == nil {
		t.Error("should fail")
	}

	// Valid
	refresh, access, _, err := ldap.AuthUser("rmiftakhov", "Belayaakula2001-")
	if err != nil {
		t.Fatal(err)
	}
	if refresh == "" || access == "" {
		t.Error("tokens empty")
	}
}

func TestValidateAccessJwt(t *testing.T) {
	config.InitConfig()

	// Get tokens from auth
	refresh, access, _, err := ldap.AuthUser("rmiftakhov", "Belayaakula2001-")
	if err != nil {
		t.Fatal(err)
	}

	// Valid access token
	u, _, err := user.ValidateAccessJwt(access)
	if err != nil {
		t.Error("should be valid")
	}
	if u.App.Username != "rmiftakhov" {
		t.Error("wrong user")
	}

	// Invalid access token
	_, _, err = user.ValidateAccessJwt("invalid")
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
	u2, _, err := user.ValidateAccessJwt(newAccess)
	if err != nil {
		t.Error("new access token should be valid")
	}
	if u2.App.Username != "rmiftakhov" {
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
	// Register new user
	refresh, access, _, err := local.RegisterUser("testuser", "testpass")
	if err != nil {
		t.Fatal(err)
	}
	if refresh == "" || access == "" {
		t.Error("tokens should not be empty after registration")
	}

	// Check user exists
	u, exists, _, err := user.GetUser("testuser")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("user should exist after registration")
	}
	if u.App.Username != "testuser" {
		t.Error("wrong username")
	}

	// Try to register again - should fail
	_, _, _, err = local.RegisterUser("testuser", "testpass")
	if err == nil {
		t.Error("registering existing user should fail")
	}
}

func TestLocalAuthUser(t *testing.T) {
	// First register
	_, _, _, err := local.RegisterUser("testuser2", "testpass")
	if err != nil {
		t.Fatal(err)
	}

	// Valid login
	refresh, access, _, err := local.AuthUser("testuser2", "testpass")
	if err != nil {
		t.Fatal(err)
	}
	if refresh == "" || access == "" {
		t.Error("tokens should not be empty")
	}

	// Invalid password
	_, _, _, err = local.AuthUser("testuser2", "wrongpass")
	if err == nil {
		t.Error("invalid password should fail")
	}

	// Non-existent user
	_, _, _, err = local.AuthUser("nonexistent", "pass")
	if err == nil {
		t.Error("non-existent user should fail")
	}
}

func TestLocalRefreshToken(t *testing.T) {
	// Register and login
	refresh, _, _, err := local.RegisterUser("testuser3", "testpass")
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
	u, _, err := user.ValidateAccessJwt(newAccess)
	if err != nil {
		t.Fatal(err)
	}
	if u.App.Username != "testuser3" {
		t.Error("wrong user in refreshed token")
	}
}
