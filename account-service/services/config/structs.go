package config

type Config struct {
	LDAP             LDAP
	JWT              JWT
	DB               DB
	ExternalAllowReg bool
	ExternalRegCode  string
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
