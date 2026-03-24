package user

import (
	"account-service/services/config"
	"database/sql"
	"fmt"
)

/*
add user to databse, return error
*/
func AddUser(user User) error {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Data.DB.Host, config.Data.DB.Port, config.Data.DB.User, config.Data.DB.Pass, config.Data.DB.Name)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `INSERT INTO users (username, mail, phone, first_name, second_name, password_hash, auth_type) 
              VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (username) DO NOTHING`
	err = db.QueryRow(query, user.App.Username, user.Personal.Mail, user.Personal.Phone,
		user.Data.FirstName, user.Data.SecondName, user.App.Password, user.App.AuthType).Scan(&user.App.UserId)
	return err
}

/*
get user data by username from databse, return data,bool, error (user data, exists, error)
*/
func GetUser(username string) (User, bool, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Data.DB.Host, config.Data.DB.Port, config.Data.DB.User, config.Data.DB.Pass, config.Data.DB.Name)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return User{}, false, err
	}
	defer db.Close()

	var u User
	query := `SELECT id, username, mail, phone, first_name, second_name FROM users WHERE username = $1`
	row := db.QueryRow(query, username)
	err = row.Scan(&u.App.UserId, &u.App.Username, &u.Personal.Mail, &u.Personal.Phone, &u.Data.FirstName, &u.Data.SecondName)
	if err == (sql.ErrNoRows) {
		//user doesnt exists
		return User{}, false, nil
	} else if err != nil {
		//database error
		return User{}, true, err
	}
	return u, true, nil
}

func RegUser(user User) error {
	_, exists, err := GetUser(user.App.Username)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("User already exists")
	}
	user.App.AuthType = "local"
	err = AddUser(user)
	return err
}
