package messenger

/*
CREATE TABLE IF NOT EXISTS messages (

	id BIGSERIAL PRIMARY KEY,
	chat_id BIGINT REFERENCES chats(id) ON DELETE CASCADE,
	sender_id INTEGER NOT NULL,
	content BYTEA NOT NULL, -- encrypted message
	timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	read TIMESTAMP

);
*/
type Message struct {
	MessageId int64  `json:"message_id" db:"id"`
	ChatId    int64  `json:"chat_id" db:"chat_id"`
	SenderId  int64  `json:"sender_id" db:"sender_id"`
	Content   string `json:"encrypted_messege" db:"content"`
	CreatedAt int64  `json:"created_at" db:"timestamp"` //unix timestamp
	Read      int64  `json:"read_time" db:"read"`       //unix timestmap
}

/*
CREATE TABLE IF NOT EXISTS rsa_keys (

	id BIGSERIAL PRIMARY KEY,
	user_id INTEGER NOT NULL,
	pub_key TEXT NOT NULL,
	priv_key TEXT NOT NULL, -- encrypted priv PEM
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

);
*/
type KeyChain struct {
	KeyChainId int64  `json:"key_chain_id" db:"-"`
	UserId     int64  `json:"user_id" db:"user_id"`
	PubKey     string `json:"pub_key" db:"pub_key"`
	PrivKey    string `json:"encrypted_priv_key" db:"priv_key"`
	CreatedAt  int64  `json:"created_at" db:"-"` //unix timestamp
}

/*
CREATE TABLE IF NOT EXISTS chats (

	id BIGSERIAL PRIMARY KEY,
	user1_id INTEGER NOT NULL,
	user2_id INTEGER NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	CHECK (user1_id < user2_id),
	UNIQUE(user1_id, user2_id)

);
*/
type Chats struct {
	ChatId    int64 `json:"chat_id" db:"id"`
	User1     int64 `json:"user1_id" db:"user1_id"`
	User2     int64 `json:"user2_id" db:"user2_id"`
	CreatedAt int64 `json:"created_at" db:"created_at"` //unix timestamp
	UpdatedAt int64 `json:"updated_at" db:"updated_at"` //unix timestamp
}
