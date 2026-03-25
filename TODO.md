# Account Service TODO

## Authentication Handlers
- [ ] Implement POST /login handler in handlers/auth.go
  - Validate request body (username, password)
  - Call ldap.AuthUser()
  - Return tokens or error
- [ ] Implement POST /refresh-token handler
  - Validate refresh token
  - Call user.ValidateRefreshToken()
  - Return new access token
- [ ] Implement GET /profile handler
  - Validate access token
  - Return user profile data

## Middleware Integration
- [ ] Apply JWT middleware to protected routes (/refresh-token, /profile)
- [ ] Add permissions middleware if needed
- [ ] Add CORS middleware for API access

## Error Handling
- [ ] Standardize error responses (JSON format)
- [ ] Add proper HTTP status codes
- [ ] Add request validation middleware

## Security
- [ ] Ensure all endpoints use HTTPS in production
- [ ] Add rate limiting for auth endpoints
- [ ] Implement token blacklisting for logout
- [ ] Add input sanitization

## Database
- [ ] Add database migrations for schema changes
- [ ] Implement connection pooling
- [ ] Add database health checks

## Testing
- [ ] Add integration tests with real database
- [ ] Add tests for handlers
- [ ] Add tests for middleware
- [ ] Add load testing

## Deployment
- [ ] Create Dockerfile
- [ ] Add docker-compose for full stack
- [ ] Add environment configuration
- [ ] Add health check endpoint

## Documentation
- [ ] Add API documentation (OpenAPI/Swagger)
- [ ] Add README with setup instructions
- [ ] Add code comments and documentation

## Monitoring
- [ ] Add logging middleware
- [ ] Add metrics collection
- [ ] Add health check endpoint

## Additional Features
- [ ] Implement user registration (if needed)
- [ ] Add password reset functionality
- [ ] Add user management endpoints (admin)
- [ ] Add audit logging for auth events