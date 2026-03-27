package config

import "crypto/tls"

type Config struct {
	LDAP             LDAP
	JWT              JWT
	Databases        map[string]DB
	ExternalAllowReg bool
	ExternalRegCode  string
	Loki             Loki
	Service          Service
	Mtls             *tls.Config
	MtlsPort         string
}

type LDAP struct {
	Host     string
	Port     string
	CN       string
	DN       string
	Password string
}

type JWT struct {
	Secret             string
	AccessTokenExpire  int64 //unix
	RefreshTokenExpire int64 //unix
}

type DB struct {
	Host string
	Port string
	Name string
	User string
	Pass string
}

type Loki struct {
	Use bool
}

type Service struct {
	Port string
}

type Mtls struct {
	Port       string //port of mtls connection
	CaCrt      string //ca.crt filename
	ServiceSrt string //service.srt filename
	ServiceKey string //service.key filename
}
