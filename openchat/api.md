# OpenChat API Documentation

## Overview
This API is served via nginx proxy:
- **Account Service**: `https://dev.openchat.overhead-lines.ru/api/account/*` → internal account-service:8080/*
- **Message Service**: `https://dev.openchat.overhead-lines.ru/api/message/*` → internal message-service:8181/*

Both services use Gin router with global CORS (`Access-Control-Allow-Origin: *`).

## Response Format
**Wrapper used by most endpoints** (`RespondSuccess`/`RespondError`/`respondError`): `ServiceResponse`
```json
{
  "response": "<escaped JSON string of body>",
  "error": {
    "exists": "false"
  }
}
```
Errors:
```json
{
  "response": null,
  "error": {
    "exists": "true",
    "message": "description"
  }
}
```

**Endpoints WITHOUT wrapper (plain JSON)**:
- `GET /health` (both services): `{"status": "ok"}`

**All other endpoints** use the wrapper, including protected routes, auth handlers, internal mTLS.

## Authentication
- **Access Token**: HTTP-only, secure cookie `access_token` (TTL from config, e.g., 15 minutes).
- **Refresh Token**: Long-lived JWT returned in `refresh_token` field.
- Protected routes (`/protected/*`) use `CookieAuthMiddleware`: validates `access_token` cookie, extracts `user.User` to context.
- Login/Register set `access_token` cookie and return refresh details.
- Refresh issues new `access_token` cookie.
- Revoke uses `refresh_token` (note: DELETE requests expect JSON body).

**User Object** (inner body of `/protected/profile`):
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

## Account Service Endpoints (`/api/account/` prefix)

### Health Check
```
GET /health
```
- **Auth**: No
- **Response**: `200 {"status": "ok"}` (plain)

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
- **Responses**:
  - `200`: ServiceResponse with inner `{"refresh_token": "jwt", "expires_in": <seconds>}` + sets `access_token` cookie
  - `401`: Auth failed
  - `400`: Invalid format

### Local Login
```
POST /auth/login-local
```
- Identical to LDAP Login (uses local auth).

### Local Register
```
POST /auth/register-local
```
- Same request body.
- **Responses**:
  - `200`: ServiceResponse with inner `{"refresh_token": "jwt", "expires_in": <seconds>}` + cookie (success creates user)
  - `409`: Registration failed (e.g., user exists)
  - `400`: Invalid format

### Refresh Token
```
POST /public/refresh-token
```
- **Auth**: No
- **Body**: `{"refresh_token": "jwt"}`
- **Responses**:
  - `200`: ServiceResponse with inner `{"success": true, "expires_in": <access seconds>}` + new `access_token` cookie
  - `401`: Invalid refresh token
  - `400`: Invalid format

### Revoke Token
```
DELETE /public/revoke-token
```
- **Auth**: No
- **Body**: `{"refresh_token": "jwt"}` (note: DELETE with JSON body)
- **Responses**:
  - `200`: ServiceResponse with inner `{"success": true}`
  - `500`: Failed to revoke
  - `400`: Invalid format

### Revoke All Tokens
```
DELETE /public/revoke-all-tokens
```
- Identical to Revoke Token.

### Profile
```
GET /protected/profile
```
- **Auth**: Yes (cookie)
- **Responses**: `200` ServiceResponse with inner User object

### Internal Service Auth (mTLS)
```
POST /api/account/service/verify-access-token
```
- **Internal only** (mTLS)
- **Body**: `{"access_token": "jwt"}`
- **Responses**: `200` ServiceResponse with inner User object
- `401`: Invalid token

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
- Generates mnemonic + private key if none exists for user.
- **Responses**:
  - `200`: ServiceResponse with inner `{"words": ["word1", ...], "private_key": "string"}`
  - `400`: Keys already exist or invalid auth/user
  - `500`: Generation/DB error

## Middleware Summary
- **CookieAuthMiddleware** (both services): Validates `access_token` cookie → `user.ValidateAccessJwt()` → `c.Set("user", user.User)`.
- Duplicate impl in `middleware/message-service/service.go`.
- No permissions middleware applied currently.