package user

type User struct {
	Data     Data     `json:"data"`
	Personal Personal `json:"personal"`
	App      App      `json:"app"`
}

type Personal struct {
	Mail  string `json:"mail" db:"mail"`
	Phone string `json:"phone" db:"phone"`
}

type Data struct {
	FirstName  string `json:"firstName" db:"first_name"`
	SecondName string `json:"secondName" db:"second_name"`
}

type App struct {
	UserId   int64  `json:"userId" db:"id"`
	Username string `json:"username" db:"username"`
	Password string `json:"password" db:"password_hash"`
	AuthType string `json:"authType" db:"auth_type"`
}
