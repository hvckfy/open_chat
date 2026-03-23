package config

import (
	"account-service/services/environment"
	"fmt"
	"reflect"
)

/*
Initializate config
*/
func initConfig() Config {
	cfg := Config{
		LDAPHost: environment.GetEnvValue("LDAPHost", "9.9.9.17"),
		LDAPPort: environment.GetEnvValue("LDAPPort", "389"),
		LDAP: LDAPServiceAccount{
			CN:       environment.GetEnvValue("LDAPServiceAccountDN", "cn=admin"),
			DN:       environment.GetEnvValue("LDAPServiceAccountDC", "dc=heckfy,dc=local"),
			Password: environment.GetEnvValue("LDAPServiceAccountPassword", "admin"),
		},
	}

	return cfg
}

// Store config
var Data = initConfig()
var reflectedData = reflect.ValueOf(Data)

/*
Get value from config buy key
*/
func GetValue(key string) (string, error) {
	field := reflectedData.FieldByName(key)
	if !field.IsValid() {
		return "", fmt.Errorf("Unknown key: %s", key)
	}
	return field.String(), nil
}
