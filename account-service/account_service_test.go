package main

import (
	"account-service/services/auth/ldap"
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
	_, _, err := ldap.AuthUser("invalid", "pass")
	if err == nil {
		t.Error("should fail")
	}

	// Valid
	refresh, access, err := ldap.AuthUser("rmiftakhov", "Belayaakula2001-")
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
	refresh, access, err := ldap.AuthUser("rmiftakhov", "Belayaakula2001-")
	if err != nil {
		t.Fatal(err)
	}

	// Valid access token
	u, err := user.ValidateAccessJwt(access)
	if err != nil {
		t.Error("should be valid")
	}
	if u.App.Username != "rmiftakhov" {
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
