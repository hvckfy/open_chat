-- Chats (P2P: two users only)
CREATE TABLE IF NOT EXISTS chats (
    id BIGSERIAL PRIMARY KEY,
    user1_id INTEGER NOT NULL,
    user2_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CHECK (user1_id < user2_id),
    UNIQUE(user1_id, user2_id)
);

-- Messages
CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT REFERENCES chats(id) ON DELETE CASCADE,
    sender_id INTEGER NOT NULL,
    content BYTEA NOT NULL, -- encrypted message
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    read TIMESTAMP
);



CREATE INDEX IF NOT EXISTS idx_chat_timestamp ON messages(chat_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_sender_timestamp ON messages(sender_id, timestamp DESC);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_chats_updated_at BEFORE UPDATE ON chats
FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- RSA keys per user (priv encrypted)
CREATE TABLE IF NOT EXISTS rsa_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    pub_key TEXT NOT NULL,
    priv_key TEXT NOT NULL, -- encrypted priv PEM
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rsa_user ON rsa_keys(user_id);
