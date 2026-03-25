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

	accessToken, err := strconv.ParseInt(environment.GetEnvValue("AccessTokenExpire", "60"), 10, 64)
	refreshToken, err := strconv.ParseInt(environment.GetEnvValue("RefreshTokenExpire", "3600"), 10, 64)
	if err != nil {
		return fmt.Errorf("AccessTokenExpire and RefreshTokenExpire must be int64 values")
	}

	externalReg, err := strconv.ParseBool(environment.GetEnvValue("ExternalAllowReg", "true"))
	if err != nil {
		return fmt.Errorf("EternalReg must be boolean value")
	}

	lokiUse, err := strconv.ParseBool(environment.GetEnvValue("LokiUse", "true"))
	if err != nil {
		return fmt.Errorf("LokiUse must be boolean value")
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
		DB: DB{
			Host: environment.GetEnvValue("DBHost", "9.9.9.17"),
			Port: environment.GetEnvValue("DBPort", "5432"),
			Name: environment.GetEnvValue("DBName", "accountdb"),
			User: environment.GetEnvValue("DBUser", "accountuser"),
			Pass: environment.GetEnvValue("DBPass", "accountpass"),
		},
		ExternalAllowReg: externalReg,
		ExternalRegCode:  environment.GetEnvValue("ExternalRegCode", "registration_code_for_external_people"),
		Loki: Loki{
			Use: lokiUse,
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
