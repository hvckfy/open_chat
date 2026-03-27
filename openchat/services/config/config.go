package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"openchat/meta"
	"os"
	"reflect"
	"strconv"
)

var Data Config
var reflectedData reflect.Value

/*
Initializate config
*/
func InitAccountServiceConfig() error {

	Mtls := Mtls{
		Port:       meta.GetEnvValue("MTLSPort", "48080"),
		CaCrt:      meta.GetEnvValue("MTLSCaCrt", "/home/certs/ca.crt"),
		ServiceSrt: meta.GetEnvValue("MTLSServiceCrt", "/home/certs/account-service.crt"),
		ServiceKey: meta.GetEnvValue("MTLSServiceKey", "/home/certs/account-service.key"),
	}

	cert, err := tls.LoadX509KeyPair(
		Mtls.ServiceSrt,
		Mtls.ServiceKey,
	)
	if err != nil {
		return err
	}
	caCert, err := os.ReadFile(Mtls.CaCrt)
	if err != nil {
		return err
	}
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		ClientCAs:          caPool,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		RootCAs:            caPool, // доверяем серверу
		InsecureSkipVerify: false,  // проверяем сервер
	}

	accessToken, err := strconv.ParseInt(meta.GetEnvValue("AccessTokenExpire", "60"), 10, 64)
	refreshToken, err := strconv.ParseInt(meta.GetEnvValue("RefreshTokenExpire", "3600"), 10, 64)
	if err != nil {
		return fmt.Errorf("AccessTokenExpire and RefreshTokenExpire must be int64 values")
	}

	externalReg, err := strconv.ParseBool(meta.GetEnvValue("ExternalAllowReg", "true"))
	if err != nil {
		return fmt.Errorf("EternalReg must be boolean value")
	}

	lokiUse, err := strconv.ParseBool(meta.GetEnvValue("LokiUse", "true"))
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
			Port: meta.GetEnvValue("ServicePort", "8080"),
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
		Mtls:     tlsConfig,
		MtlsPort: Mtls.Port,
	}
	return nil
}

func InitMessageServiceConfig() error {

	Mtls := Mtls{
		//Port:       meta.GetEnvValue("MTLSPort", "48080"), //host port
		CaCrt:      meta.GetEnvValue("MTLSCaCrt", "/home/certs/ca.crt"),
		ServiceSrt: meta.GetEnvValue("MTLSServiceCrt", "/home/certs/message-service.crt"),
		ServiceKey: meta.GetEnvValue("MTLSServiceKey", "/home/certs/message-service.key"),
	}

	cert, err := tls.LoadX509KeyPair(
		Mtls.ServiceSrt,
		Mtls.ServiceKey,
	)
	if err != nil {
		return err
	}
	caCert, err := os.ReadFile(Mtls.CaCrt)
	if err != nil {
		return err
	}
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		ClientCAs:          caPool,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		RootCAs:            caPool, // доверяем серверу
		InsecureSkipVerify: false,  // проверяем сервер
	}

	lokiUse, err := strconv.ParseBool(meta.GetEnvValue("LokiUse", "true"))
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
		Mtls: tlsConfig,
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
