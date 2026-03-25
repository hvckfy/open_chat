package ldap

import (
	"fmt"
	"openchat/services/auth/user"
	"openchat/services/config"
	"openchat/services/logger"

	"github.com/go-ldap/ldap/v3"
	"go.uber.org/zap"
)

/*
Verify user credentials against LDAP server
*/
func VerifyUser(username string, password string) (bool, error) {
	logger.Info("LDAP VerifyUser called",
		zap.String("username", username),
		zap.String("server", config.Data.LDAP.Host))

	LDAPBaseDn := config.Data.LDAP.DN
	LDAPCn := config.Data.LDAP.CN
	LDAPUrl := fmt.Sprintf("ldap://%s:%s", config.Data.LDAP.Host, config.Data.LDAP.Port)
	LDAPpassword := config.Data.LDAP.Password

	// 1. Connect to LDAP server
	l, err := ldap.DialURL(LDAPUrl)
	if err != nil {
		logger.Error("LDAP connection failed",
			zap.String("url", LDAPUrl),
			zap.Error(err))
		return false, fmt.Errorf("LDAP connection failed: %w", err)
	}
	defer l.Close()

	// 2. Service Bind
	err = l.Bind(fmt.Sprintf("%s,%s", LDAPCn, LDAPBaseDn), LDAPpassword)
	if err != nil {
		logger.Error("LDAP service bind failed",
			zap.String("bind_dn", fmt.Sprintf("%s,%s", LDAPCn, LDAPBaseDn)),
			zap.Error(err))
		return false, fmt.Errorf("service bind failed: %w", err)
	}

	// 3. Search for the user's DN
	searchRequest := ldap.NewSearchRequest(
		LDAPBaseDn,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=person)(uid=%s))", ldap.EscapeFilter(username)),
		[]string{"dn"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		logger.Error("LDAP user search failed",
			zap.String("username", username),
			zap.String("base_dn", LDAPBaseDn),
			zap.Error(err))
		return false, fmt.Errorf("user search failed: %w", err)
	}

	if len(sr.Entries) != 1 {
		logger.Warn("LDAP user not found or multiple entries",
			zap.String("username", username),
			zap.Int("entries_found", len(sr.Entries)))
		return false, fmt.Errorf("user not found or multiple entries returned")
	}

	userDN := sr.Entries[0].DN

	// 4. User Bind (Verify password)
	err = l.Bind(userDN, password)
	if err != nil {
		logger.Warn("LDAP user bind failed (invalid credentials)",
			zap.String("username", username),
			zap.String("user_dn", userDN))
		return false, fmt.Errorf("invalid credentials")
	}

	logger.Info("LDAP user verification successful",
		zap.String("username", username))
	return true, nil
}

/*
Import user data from LDAP to database
*/
func ImportUser(username string) (bool, error) {
	logger.Info("LDAP ImportUser called",
		zap.String("username", username))

	LDAPBaseDn := config.Data.LDAP.DN
	LDAPCn := config.Data.LDAP.CN
	LDAPUrl := fmt.Sprintf("ldap://%s:%s", config.Data.LDAP.Host, config.Data.LDAP.Port)
	LDAPpassword := config.Data.LDAP.Password

	// Connect to LDAP
	l, err := ldap.DialURL(LDAPUrl)
	if err != nil {
		logger.Error("LDAP connection failed for import",
			zap.String("username", username),
			zap.Error(err))
		return false, fmt.Errorf("LDAP connection failed: %w", err)
	}
	defer l.Close()

	// Service Bind
	err = l.Bind(fmt.Sprintf("%s,%s", LDAPCn, LDAPBaseDn), LDAPpassword)
	if err != nil {
		logger.Error("LDAP service bind failed for import",
			zap.String("username", username),
			zap.Error(err))
		return false, fmt.Errorf("service bind failed: %w", err)
	}

	// Search for user's DN
	searchRequest := ldap.NewSearchRequest(
		LDAPBaseDn,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=person)(uid=%s))", ldap.EscapeFilter(username)),
		[]string{"dn"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		logger.Error("LDAP user search failed for import",
			zap.String("username", username),
			zap.Error(err))
		return false, fmt.Errorf("user search failed: %w", err)
	}

	if len(sr.Entries) != 1 {
		logger.Warn("LDAP user not found for import",
			zap.String("username", username))
		return false, fmt.Errorf("LDAP user not found")
	}

	userDN := sr.Entries[0].DN

	// Search for user attributes
	userSearch := ldap.NewSearchRequest(
		userDN,
		ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=*)",
		[]string{"mail", "telephoneNumber", "givenName", "sn", "ou"},
		nil,
	)

	userSr, err := l.Search(userSearch)
	if err != nil {
		logger.Error("LDAP user attributes search failed",
			zap.String("username", username),
			zap.String("user_dn", userDN),
			zap.Error(err))
		return false, fmt.Errorf("failed to get user attributes: %w", err)
	}

	if len(userSr.Entries) != 1 {
		logger.Error("LDAP user attributes not found",
			zap.String("username", username))
		return false, fmt.Errorf("failed to get user attributes")
	}

	entry := userSr.Entries[0]
	u := user.User{
		Personal: user.Personal{
			Mail:  entry.GetAttributeValue("mail"),
			Phone: entry.GetAttributeValue("telephoneNumber"),
		},
		Data: user.Data{
			FirstName:  entry.GetAttributeValue("givenName"),
			SecondName: entry.GetAttributeValue("sn"),
		},
		App: user.App{
			Username: username,
			AuthType: "ldap",
		},
	}

	u, err = user.AddUser(u)
	if err != nil {
		logger.Error("Failed to add LDAP user to database",
			zap.String("username", username),
			zap.Error(err))
		return false, fmt.Errorf("failed to add user: %w", err)
	}

	logger.Info("LDAP user imported successfully",
		zap.String("username", username),
		zap.Int64("user_id", u.App.UserId))
	return true, nil
}

/*
Authenticate user via LDAP and return JWT tokens
*/
func AuthUser(username string, password string) (string, string, error) {
	logger.Info("LDAP AuthUser called",
		zap.String("username", username))

	// Verify credentials against LDAP
	valid, err := VerifyUser(username, password)
	if err != nil {
		logger.Error("LDAP authentication failed",
			zap.String("username", username),
			zap.Error(err))
		return "", "", fmt.Errorf("LDAP authentication failed: %w", err)
	}

	if !valid {
		logger.Warn("LDAP authentication failed: invalid credentials",
			zap.String("username", username))
		return "", "", fmt.Errorf("invalid credentials")
	}

	// Check if user exists in database
	u, exists, err := user.GetUser(username)
	if err != nil {
		logger.Error("Failed to get user from database",
			zap.String("username", username),
			zap.Error(err))
		return "", "", fmt.Errorf("failed to get user: %w", err)
	}

	// Import user if not exists
	if !exists {
		logger.Info("User not found in database, importing from LDAP",
			zap.String("username", username))

		success, err := ImportUser(username)
		if err != nil {
			logger.Error("Failed to import LDAP user",
				zap.String("username", username),
				zap.Error(err))
			return "", "", fmt.Errorf("failed to import user: %w", err)
		}
		if !success {
			logger.Error("LDAP user import failed",
				zap.String("username", username))
			return "", "", fmt.Errorf("failed to add LDAP user")
		}

		// Get user again after import
		u, exists, err = user.GetUser(username)
		if err != nil {
			logger.Error("Failed to get imported user",
				zap.String("username", username),
				zap.Error(err))
			return "", "", fmt.Errorf("failed to get imported user: %w", err)
		}
		if !exists {
			logger.Error("Imported user not found",
				zap.String("username", username))
			return "", "", fmt.Errorf("imported user not found")
		}
	}

	// Generate tokens
	refreshToken, refreshExpireAt, err := user.GenerateJwt(username, config.Data.JWT.RefreshTokenExpire)
	if err != nil {
		logger.Error("Failed to generate refresh token",
			zap.String("username", username),
			zap.Error(err))
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	accessToken, _, err := user.GenerateJwt(username, config.Data.JWT.AccessTokenExpire)
	if err != nil {
		logger.Error("Failed to generate access token",
			zap.String("username", username),
			zap.Error(err))
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Store refresh token
	success, err := user.AddRefreshJwt(u.App.UserId, refreshToken, refreshExpireAt)
	if err != nil {
		logger.Error("Failed to store refresh token",
			zap.String("username", username),
			zap.Error(err))
		return "", "", fmt.Errorf("failed to store refresh token: %w", err)
	}
	if !success {
		logger.Error("Failed to register refresh token",
			zap.String("username", username))
		return "", "", fmt.Errorf("failed to register refresh token")
	}

	logger.Info("LDAP authentication successful",
		zap.String("username", username),
		zap.Int64("user_id", u.App.UserId))
	return refreshToken, accessToken, nil
}
