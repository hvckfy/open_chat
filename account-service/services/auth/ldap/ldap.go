package ldap

import (
	"account-service/services/auth/user"
	"account-service/services/config"
	"account-service/services/errofy"
	"fmt"

	"github.com/go-ldap/ldap/v3"
	_ "github.com/lib/pq"
)

/*
Auth user log/pass from ldap return verify, error_code, error
*/
func VerifyUser(username string, password string) (bool, int64, error) {
	LDAPBaseDn := config.Data.LDAP.DN
	LDAPCn := config.Data.LDAP.CN
	LDAPUrl := fmt.Sprintf("ldap://%s:%s", config.Data.LDAP.Host, config.Data.LDAP.Port)
	LDAPpassword := config.Data.LDAP.Password
	// 1. Connect to LDAP server
	l, err := ldap.DialURL(LDAPUrl)
	if err != nil {
		errofy.LogError(5011, err, "VerifyUser")
		return false, 5011, err
	}
	defer l.Close()

	// 2. Service Bind (Admin/Service account login)
	err = l.Bind(fmt.Sprintf("%s,%s", LDAPCn, LDAPBaseDn), LDAPpassword)
	if err != nil {
		errofy.LogError(5014, err, "VerifyUser")
		return false, 5014, fmt.Errorf("service bind failed: %w", err)
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
		return false, 5015, fmt.Errorf("user not found or multiple entries returned")
	}

	userDN := sr.Entries[0].DN

	// 4. User Bind (Verify password)
	err = l.Bind(userDN, password)
	if err != nil {
		return false, 5012, fmt.Errorf("invalid credentials")
	}

	return true, 200, nil
}

/*
After auth, add data to db return success, error_code, error
*/
func ImportUser(username string) (bool, int64, error) {
	LDAPBaseDn := config.Data.LDAP.DN
	LDAPCn := config.Data.LDAP.CN
	LDAPUrl := fmt.Sprintf("ldap://%s:%s", config.Data.LDAP.Host, config.Data.LDAP.Port)
	LDAPpassword := config.Data.LDAP.Password

	// Connect to LDAP
	l, err := ldap.DialURL(LDAPUrl)
	if err != nil {
		errofy.LogError(5011, err, "Importuser")
		return false, 5011, err
	}
	defer l.Close()

	// Service Bind
	err = l.Bind(fmt.Sprintf("%s,%s", LDAPCn, LDAPBaseDn), LDAPpassword)
	if err != nil {
		errofy.LogError(5014, err, "ImportUser")
		return false, 5014, fmt.Errorf("service bind failed: %w", err)
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
		return false, 200, fmt.Errorf("LDAP user not found")
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
		errofy.LogError(5012, err, "ImportUser")
		return false, 5012, fmt.Errorf("failed to get user attributes")
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
	u, error_code, err := user.AddUser(u)
	if err != nil {
		return false, error_code, err
	}
	return true, error_code, nil
}

/*
Auth user via ldap. Return refresh JWT, access JWT,error_code, error
*/
func AuthUser(username string, password string) (string, string, int64, error) {
	valid, error_code, err := VerifyUser(username, password)
	if err != nil {
		return "", "", error_code, err
	}
	if valid {
		u, exists, error_code, err := user.GetUser(username)
		if err != nil {
			return "", "", error_code, err
		}
		if !exists {
			success, error_code, err := ImportUser(username)
			if err != nil {
				return "", "", error_code, err
			}
			if !success {
				return "", "", error_code, fmt.Errorf("failed to add LDAP user")
			}
		}
		u, exists, error_code, err = user.GetUser(username)
		if err != nil {
			return "", "", error_code, err
		}
		if error_code != 200 {
			return "", "", error_code, err
		}
		if !exists {
			return "", "", error_code, fmt.Errorf("Can not put tokens to unregistered user")
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
		//add tokens HERE
		success, errorCode, err := user.AddRefreshJwt(u.App.UserId, refreshToken, refreshExpireAt)
		if err != nil {
			return "", "", errorCode, err
		}
		if !success {
			return "", "", errorCode, fmt.Errorf("unsuccess token registration")
		}
		if error_code != 200 {
			return "", "", error_code, err
		}
		return refreshToken, accessToken, 200, nil
	} else {
		return "", "", error_code, fmt.Errorf("Username and password doesnt match")
	}
}
