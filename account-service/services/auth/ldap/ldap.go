package ldap

import (
	"account-service/services/config"
	"fmt"

	"github.com/go-ldap/ldap/v3"
)

/*
Auth user log/pass from ldap host
*/
func AuthUser(username, password string) (bool, error) {
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
