package main

import (
	"account-service/services/auth/ldap"
	"fmt"
)

func main() {
	fmt.Println(ldap.AuthUser("rmiftakhov", "Belayaakula2001-"))
}
