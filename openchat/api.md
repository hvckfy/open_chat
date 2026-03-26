# OpenChat API Documentation

## Overview
This API is served via nginx proxy:
- **Account Service**: `https://dev.openchat.overhead-lines.ru/api/account/*` → internal account-service:8080/*
- **Message Service**: `https://dev.openchat.overhead-lines.ru/api/message/*` → internal message-service:8181/*

Both services use Gin router with global CORS (`Access-Control-Allow-Origin: *`).

## Authentication
- **Access Token**: HTTP-only, secure cookie `access_token` (15 minutes TTL).
- **Refresh Token**: Long-lived JWT returned in responses for login/register/refresh.
- Protected routes validate `access_token` cookie, extract `user.User` to context.
- Login/Register set `access_token` cookie and return `refresh_token`.
- Refresh issues new `access_token` cookie.
- Revoke uses `refresh_token`.

**User Object** (returned by `/protected/profile`):
```json
{
  "data": {
    "firstName": "string",
    "secondName": "string"
  },
  "personal": {
    "mail": "string",
    "phone": "string"
  },
  "app": {
    "userId": 12345,
    "username": "string",
    "authType": "ldap|local"
  }
}
```

Error responses: `{"error": "description"}` (4xx/5xx).

## Account Service Endpoints (`/api/account/` prefix)

### Health Check
```
GET /health
```
- **Auth**: No
- **Response**: `200 {"status": "ok"}`

### LDAP Login
```
POST /auth/login-ldap
```
- **Auth**: No
- **Body**:
  ```json
  {
    "username": "string",
    "password": "string"
  }
  ```
- **Response**:
  - `200`: `{"refresh_token": "jwt", "expires_in": 900}` + sets `access_token` cookie
  - `401`: Authentication failed
  - `400`: Invalid format

### Local Login
```
POST /auth/login-local
```
- Same as LDAP Login.

### Local Register
```
POST /auth/register-local
```
- Same as Login.
- **Response**: `201` on success.

### Refresh Token
```
POST /public/refresh-token
```
- **Auth**: No
- **Body**: `{"refresh_token": "jwt"}`
- **Response**:
  - `200`: `{"expires_in": 900}` + new `access_token` cookie
  - `401`: Invalid refresh token

### Revoke Token
```
DELETE /public/revoke-token
```
- **Auth**: No
- **Body**: `{"refresh_token": "jwt"}`
- **Response**: `200 {"message": true}`

### Revoke All Tokens
```
DELETE /public/revoke-all-tokens
```
- Same as Revoke Token.

### Profile
```
GET /protected/profile
```
- **Auth**: Yes (cookie)
- **Response**: `200` User object

## Message Service Endpoints (`/api/message/` prefix)

### Health Check
```
GET /health
```
- Same as Account.

### Generate Keys
```
GET /protected/gen-keys
```
- **Auth**: Yes (cookie)
- Generates mnemonic words + private key for user (if none exists).
- **Response**:
  - `200`: `{"words": ["word1", "word2", ...], "private_key": "string"}`
  - `400`: Keys already exist or auth/invalid user
  - `500`: Generation/DB error

## Middleware Summary
- **CookieAuthMiddleware** (both services): Validates `access_token` cookie → `user.ValidateAccessJwt()` → sets `c.Set("user", user.User)`.
- No additional middleware (e.g., permissions) applied to routes.
- Duplicate impl in `middleware/message-service/service.go` (unused).