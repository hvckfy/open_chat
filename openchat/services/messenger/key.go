package messenger

import (
	"database/sql"
	"fmt"
	"openchat/services/config"
)

/*
Get users encryptedPrivKey and pubKey from db
*/
func GetKeys(userID int64) (encryptedPrivKey []byte, pubKey []byte, exists bool, err error) {

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Data.Databases["MessageDb"].Host,
		config.Data.Databases["MessageDb"].Port,
		config.Data.Databases["MessageDb"].User,
		config.Data.Databases["MessageDb"].Pass,
		config.Data.Databases["MessageDb"].Name,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, nil, false, err
	}
	defer db.Close()

	err = db.QueryRow(
		`SELECT pub_key, priv_key FROM rsa_keys WHERE user_id = $1`,
		userID,
	).Scan(&pubKey, &encryptedPrivKey)

	if err == sql.ErrNoRows {
		return nil, nil, false, nil
	}

	if err != nil {
		return nil, nil, false, err
	}

	return encryptedPrivKey, pubKey, true, nil
}

/*
Put users keys into database
*/
func PutKeys(user_id int64, encryptedPrivKey []byte, PubKey []byte) (success bool, err error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Data.Databases["MessageDb"].Host,
		config.Data.Databases["MessageDb"].Port,
		config.Data.Databases["MessageDb"].User,
		config.Data.Databases["MessageDb"].Pass,
		config.Data.Databases["MessageDb"].Name)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return false, err
	}
	defer db.Close()

	//insert:
	_, err = db.Exec(
		"INSERT INTO rsa_keys (user_id, pub_key, priv_key) VALUES ($1, $2, $3)",
		user_id, PubKey, encryptedPrivKey)
	if err != nil {
		return false, err
	}
	//return user words and NOT encrypted private key
	return true, nil

}
