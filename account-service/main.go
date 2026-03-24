package main

import (
	"account-service/services/auth/ldap"
	"account-service/services/config"
	"fmt"
)

func main() {
	err := config.InitConfig()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(ldap.AuthUser("rmiftakhov", "Belayaakula2001-"))
}
