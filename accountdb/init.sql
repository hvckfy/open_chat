CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    mail VARCHAR(255),
    phone VARCHAR(255),
    first_name VARCHAR(255),
    second_name VARCHAR(255),
    username VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    password_hash VARCHAR(255),
    auth_type VARCHAR(10)
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);