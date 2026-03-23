package config

type Config struct {
	LDAP     LDAPServiceAccount
	LDAPHost string
	LDAPPort string
}

type LDAPServiceAccount struct {
	CN       string
	DN       string
	Password string
}
