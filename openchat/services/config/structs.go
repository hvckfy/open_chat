package config

type Config struct {
	LDAP             LDAP
	JWT              JWT
	Databases        map[string]DB
	ExternalAllowReg bool
	ExternalRegCode  string
	Loki             Loki
	Service          Service
	InternalServices map[string]InternalService
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
	Port              string
	AuthentifyPrivKey string //filename of <name>service-private.pem for authentify (accepiting)
}

type InternalService struct {
	Host             string
	Port             string
	AuthentifyPubKey string //filename of internalservice-public.pem for authentify (requestiong)
}
