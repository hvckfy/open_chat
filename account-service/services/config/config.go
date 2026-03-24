package config

import (
	"account-service/services/environment"
	"fmt"
	"reflect"
	"strconv"
)

var Data Config
var reflectedData reflect.Value

/*
Initializate config
*/
func InitConfig() error {

	accessToken, err := strconv.ParseInt(environment.GetEnvValue("AccessTokenExpire", "secretkey"), 10, 64)
	refreshToken, err := strconv.ParseInt(environment.GetEnvValue("RefreshTokenExpire", "secretkey"), 10, 64)
	if err != nil {
		return fmt.Errorf("AccessTokenExpire and RefreshTokenExpire must be int64 values")
	}

	Data = Config{
		LDAP: LDAP{
			Host:     environment.GetEnvValue("LDAPHost", "9.9.9.17"),
			Port:     environment.GetEnvValue("LDAPPort", "389"),
			CN:       environment.GetEnvValue("LDAPServiceAccountDN", "cn=admin"),
			DN:       environment.GetEnvValue("LDAPServiceAccountDC", "dc=heckfy,dc=local"),
			Password: environment.GetEnvValue("LDAPServiceAccountPassword", "admin"),
		},
		JWT: JWT{
			Secret:             environment.GetEnvValue("JWTSecret", "secretkey"),
			AccessTokenExpire:  accessToken,
			RefreshTokenExpire: refreshToken,
		},
	}
	return nil
}

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
