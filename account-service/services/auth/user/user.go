package user

import (
	"account-service/services/config"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // PostgreSQL driver
)

/*
add user to database, return user, error
*/
func AddUser(user User) (User, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Data.DB.Host, config.Data.DB.Port, config.Data.DB.User, config.Data.DB.Pass, config.Data.DB.Name)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return User{}, fmt.Errorf("database connection failed: %w", err)
	}
	defer db.Close()

	query := `INSERT INTO users (username, mail, phone, first_name, second_name, password_hash, auth_type)
              VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (username) DO NOTHING RETURNING id`
	err = db.QueryRow(query, user.App.Username, user.Personal.Mail, user.Personal.Phone,
		user.Data.FirstName, user.Data.SecondName, user.App.Password, user.App.AuthType).Scan(&user.App.UserId)
	if err != nil {
		return User{}, fmt.Errorf("failed to create user %s: %w", user.App.Username, err)
	}

	return user, nil
}

/*
get user data by username from database, return user, exists, error
*/
func GetUser(username string) (User, bool, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Data.DB.Host, config.Data.DB.Port, config.Data.DB.User, config.Data.DB.Pass, config.Data.DB.Name)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return User{}, false, fmt.Errorf("database connection failed: %w", err)
	}
	defer db.Close()

	var u User
	query := `SELECT id, username, mail, phone, first_name, second_name, password_hash FROM users WHERE username = $1`
	row := db.QueryRow(query, username)
	err = row.Scan(&u.App.UserId, &u.App.Username, &u.Personal.Mail, &u.Personal.Phone, &u.Data.FirstName, &u.Data.SecondName, &u.App.Password)
	if err == sql.ErrNoRows {
		return User{}, false, nil // User not found - not an error
	} else if err != nil {
		return User{}, false, fmt.Errorf("database query failed for user %s: %w", username, err)
	}
	return u, true, nil
}
