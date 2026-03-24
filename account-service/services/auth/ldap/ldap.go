package ldap

import (
	"account-service/services/auth/user"
	"account-service/services/config"
	"fmt"

	"github.com/go-ldap/ldap/v3"
	_ "github.com/lib/pq"
)

/*
Auth user log/pass from ldap host
*/
func VerifyUser(username string, password string) (bool, error) {
	LDAPBaseDn := config.Data.LDAP.DN
	LDAPCn := config.Data.LDAP.CN
	LDAPUrl := fmt.Sprintf("ldap://%s:%s", config.Data.LDAP.Host, config.Data.LDAP.Port)
	LDAPpassword := config.Data.LDAP.Password
	// 1. Connect to LDAP server
	l, err := ldap.DialURL(LDAPUrl)
	if err != nil {
		return false, err
	}
	defer l.Close()

	// 2. Service Bind (Admin/Service account login)
	err = l.Bind(fmt.Sprintf("%s,%s", LDAPCn, LDAPBaseDn), LDAPpassword)
	if err != nil {
		return false, fmt.Errorf("service bind failed: %w", err)
	}

	// 3. Search for the user's DN
	searchRequest := ldap.NewSearchRequest(
		LDAPBaseDn,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=person)(uid=%s))", ldap.EscapeFilter(username)), // Filter
		[]string{"dn"}, // Attributes to retrieve
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil || len(sr.Entries) != 1 {
		return false, fmt.Errorf("user not found or multiple entries returned")
	}

	userDN := sr.Entries[0].DN

	// 4. User Bind (Verify password)
	err = l.Bind(userDN, password)
	if err != nil {
		return false, fmt.Errorf("invalid credentials")
	}

	return true, nil
}

/*
After auth, add data to db
*/
func ImportUser(username string) (bool, error) {
	LDAPBaseDn := config.Data.LDAP.DN
	LDAPCn := config.Data.LDAP.CN
	LDAPUrl := fmt.Sprintf("ldap://%s:%s", config.Data.LDAP.Host, config.Data.LDAP.Port)
	LDAPpassword := config.Data.LDAP.Password

	// Connect to LDAP
	l, err := ldap.DialURL(LDAPUrl)
	if err != nil {
		return false, err
	}
	defer l.Close()

	// Service Bind
	err = l.Bind(fmt.Sprintf("%s,%s", LDAPCn, LDAPBaseDn), LDAPpassword)
	if err != nil {
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
	if err != nil || len(sr.Entries) != 1 {
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
	if err != nil || len(userSr.Entries) != 1 {
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
			Password: "",
			AuthType: "ldap",
		},
	}
	err = user.AddUser(u)
	if err != nil {
		return false, err
	}
	return true, nil
}

/*
Auth user via ldap. Return refresh JWT, access JWT, error
*/
func AuthUser(username string, password string) (string, string, error) {
	valid, err := VerifyUser(username, password)
	if err != nil {
		return "", "", err
	}
	if valid {
		u, exists, err := user.GetUser(username)
		if err != nil {
			return "", "", err
		}
		if !exists {
			success, err := ImportUser(username)
			if err != nil {
				return "", "", err
			}
			if !success {
				return "", "", fmt.Errorf("failed to add LDAP user")
			}
		}
		refreshToken, refreshExpireAt, err := user.GenerateJwt(username, config.Data.JWT.RefreshTokenExpire)
		if err != nil {
			return "", "", err
		}
		accessToken, _, err := user.GenerateJwt(username, config.Data.JWT.AccessTokenExpire)
		if err != nil {
			return "", "", err
		}
		//add tokens HERE
		success, err := user.AddRefreshJwt(u.App.UserId, refreshToken, refreshExpireAt)
		if err != nil {
			return "", "", err
		}
		if !success {
			return "", "", fmt.Errorf("unsuccess token registration")
		}
		return refreshToken, accessToken, nil
	} else {
		return "", "", fmt.Errorf("Username and password doesnt match")
	}
}
