package config

import (
	"fmt"
	"openchat/meta"
	"reflect"
	"strconv"
)

var Data Config
var reflectedData reflect.Value

/*
Initializate config
*/
func InitAccountServiceConfig() error {

	accessToken, err := strconv.ParseInt(meta.GetEnvValue("AccessTokenExpire", "60"), 10, 64)
	refreshToken, err := strconv.ParseInt(meta.GetEnvValue("RefreshTokenExpire", "3600"), 10, 64)
	if err != nil {
		return fmt.Errorf("AccessTokenExpire and RefreshTokenExpire must be int64 values")
	}

	externalReg, err := strconv.ParseBool(meta.GetEnvValue("ExternalAllowReg", "true"))
	if err != nil {
		return fmt.Errorf("EternalReg must be boolean value")
	}

	lokiUse, err := strconv.ParseBool(meta.GetEnvValue("LokiUse", "false"))
	if err != nil {
		return fmt.Errorf("LokiUse must be boolean value")
	}

	AccountDb := DB{
		Host: meta.GetEnvValue("DBHost", "9.9.9.17"),
		Port: meta.GetEnvValue("DBPort", "5432"),
		Name: meta.GetEnvValue("DBName", "accountdb"),
		User: meta.GetEnvValue("DBUser", "accountuser"),
		Pass: meta.GetEnvValue("DBPass", "accountpass"),
	}
	//------------
	Data = Config{
		Service: Service{
			Port:              meta.GetEnvValue("ServicePort", "8080"),
			AuthentifyPrivKey: meta.GetEnvValue("ServicePrivateKeyPath", "/Users/heckfy/Documents/openchat/rsa/accountservice.private.pem"),
		},
		LDAP: LDAP{
			Host:     meta.GetEnvValue("LDAPHost", "9.9.9.17"),
			Port:     meta.GetEnvValue("LDAPPort", "389"),
			CN:       meta.GetEnvValue("LDAPServiceAccountDN", "cn=admin"),
			DN:       meta.GetEnvValue("LDAPServiceAccountDC", "dc=heckfy,dc=local"),
			Password: meta.GetEnvValue("LDAPServiceAccountPassword", "admin"),
		},
		JWT: JWT{
			Secret:             meta.GetEnvValue("JWTSecret", "secretkey"),
			AccessTokenExpire:  accessToken,
			RefreshTokenExpire: refreshToken,
		},
		Databases: map[string]DB{
			"AccountDb": AccountDb,
		},
		ExternalAllowReg: externalReg,
		ExternalRegCode:  meta.GetEnvValue("ExternalRegCode", "registration_code_for_external_people"),
		Loki: Loki{
			Use: lokiUse,
		},
	}
	return nil
}

func InitMessageServiceConfig() error {
	lokiUse, err := strconv.ParseBool(meta.GetEnvValue("LokiUse", "false"))
	if err != nil {
		return fmt.Errorf("LokiUse must be boolean value")
	}
	MessageDb := DB{
		Host: meta.GetEnvValue("DBHost", "9.9.9.17"),
		Port: meta.GetEnvValue("DBPort", "5433"),
		Name: meta.GetEnvValue("DBName", "messagedb"),
		User: meta.GetEnvValue("DBUser", "messageuser"),
		Pass: meta.GetEnvValue("DBPass", "messagepass"),
	}
	//------------
	Data = Config{
		Service: Service{
			Port: meta.GetEnvValue("ServicePort", "8181"),
		},
		Loki: Loki{
			Use: lokiUse,
		},
		Databases: map[string]DB{
			"MessageDb": MessageDb,
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
