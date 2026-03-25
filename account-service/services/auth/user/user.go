package user

import (
	"account-service/services/config"
	"account-service/services/errofy"
	"database/sql"
	"fmt"
)

/*
add user to database, return user, error_code, error
*/
func AddUser(user User) (User, int64, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Data.DB.Host, config.Data.DB.Port, config.Data.DB.User, config.Data.DB.Pass, config.Data.DB.Name)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		errofy.LogError(5001, err, "AddUser")
		return User{}, 5001, err
	}
	defer db.Close()

	query := `INSERT INTO users (username, mail, phone, first_name, second_name, password_hash, auth_type)
              VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (username) DO NOTHING RETURNING id`
	err = db.QueryRow(query, user.App.Username, user.Personal.Mail, user.Personal.Phone,
		user.Data.FirstName, user.Data.SecondName, user.App.Password, user.App.AuthType).Scan(&user.App.UserId)
	if err != nil {
		errofy.LogError(5002, err, "AddUser")
		return User{}, 5002, err
	}
	return user, 200, nil
}

/*
get user data by username from databse, return data,bool, error (user data, exists,error_code, error)
*/
func GetUser(username string) (User, bool, int64, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Data.DB.Host, config.Data.DB.Port, config.Data.DB.User, config.Data.DB.Pass, config.Data.DB.Name)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		errofy.LogError(5001, err, "GetUser")
		return User{}, false, 5001, err
	}
	defer db.Close()

	var u User
	query := `SELECT id, username, mail, phone, first_name, second_name, password_hash FROM users WHERE username = $1`
	row := db.QueryRow(query, username)
	err = row.Scan(&u.App.UserId, &u.App.Username, &u.Personal.Mail, &u.Personal.Phone, &u.Data.FirstName, &u.Data.SecondName, &u.App.Password)
	if err == (sql.ErrNoRows) {
		//user doesnt exists
		return User{}, false, 4041, nil
	} else if err != nil {
		//database error
		errofy.LogError(5002, err, "GetUser")
		return User{}, true, 5002, err
	}
	return u, true, 200, nil
}
