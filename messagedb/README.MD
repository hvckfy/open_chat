# messagedb

**RSA keys**:
```go
var pubKey, privKey string
err := db.QueryRow(context.Background(), 
  "SELECT pub_key, priv_key FROM rsa_keys WHERE user_id = $1", userID).
  Scan(&pubKey, &privKey)
// decrypt privKey
```

**Get chat**:
```go
var chatID int64
err := db.QueryRow(context.Background(), 
  "SELECT id FROM chats WHERE user1_id = LEAST($1,$2) AND user2_id = GREATEST($1,$2)", u1, u2).Scan(&chatID)
```

**Send msg**:
```go
_, err := db.Exec(context.Background(), 
  "INSERT INTO messages (chat_id, sender_id, content) VALUES ($1, $2, $3)", 
  chatID, senderID, encryptedContent)
```

**Get messages**:
```go
rows, err := db.Query(context.Background(), 
  "SELECT id, sender_id, content, timestamp FROM messages WHERE chat_id = $1 ORDER BY timestamp DESC LIMIT 50", chatID)
defer rows.Close()
for rows.Next() {
  var id int64; var senderID int; var content []byte; var ts time.Time
  rows.Scan(&id, &senderID, &content, &ts)
  // decrypt content
}
```

**User chats**:
```go
rows, err := db.Query(context.Background(), 
  "SELECT id FROM chats WHERE user1_id = $1 OR user2_id = $1 ORDER BY updated_at DESC", userID)
```