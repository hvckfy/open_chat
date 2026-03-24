package user

type User struct {
}

type Personal struct {
	mail  string
	phone string
}

type Data struct {
	firstName  string
	secondName string
}

type App struct {
	username string
}

type System struct {
	JWT string
}

/**
USER:
	User-Personal information:
		mail
		phone
	User-Data information:
		name
		second name
	User-App information:
		username
**/
