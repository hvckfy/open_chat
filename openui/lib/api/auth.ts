import { api, setRefreshToken, getRefreshToken, clearRefreshToken } from './client';
import type {
  LoginRequest,
  RegisterRequest,
  AuthResponse,
  RefreshTokenResponse,
  RevokeTokenResponse,
  ProfileResponse,
  GenerateKeysResponse,
  HealthResponse,
} from './types';

// Health checks
export async function checkAccountHealth() {
  return api.get<HealthResponse>('/account/health');
}

export async function checkMessageHealth() {
  return api.get<HealthResponse>('/message/health');
}

// LDAP Login
export async function loginLdap(credentials: LoginRequest) {
  const result = await api.post<AuthResponse>('/account/auth/login-ldap', credentials, {
    skipAuth: true,
  });

  if (result.success) {
    setRefreshToken(result.data.refresh_token);
  }

  return result;
}

// Local Login
export async function loginLocal(credentials: LoginRequest) {
  const result = await api.post<AuthResponse>('/account/auth/login-local', credentials, {
    skipAuth: true,
  });

  if (result.success) {
    setRefreshToken(result.data.refresh_token);
  }

  return result;
}

// Local Register
export async function registerLocal(userData: RegisterRequest) {
  const result = await api.post<AuthResponse>('/account/auth/register-local', userData, {
    skipAuth: true,
  });

  if (result.success) {
    setRefreshToken(result.data.refresh_token);
  }

  return result;
}

// Refresh Token
export async function refreshToken() {
  const currentToken = getRefreshToken();
  if (!currentToken) {
    return { success: false as const, error: 'No refresh token available' };
  }

  const result = await api.post<RefreshTokenResponse>(
    '/account/public/refresh-token',
    { refresh_token: currentToken },
    { skipAuth: true }
  );

  return result;
}

// Revoke Token (Logout)
export async function revokeToken() {
  const currentToken = getRefreshToken();
  if (!currentToken) {
    return { success: false as const, error: 'No refresh token available' };
  }

  const result = await api.delete<RevokeTokenResponse>(
    '/account/public/revoke-token',
    { refresh_token: currentToken },
    { skipAuth: true }
  );

  // Always clear local token on logout attempt
  clearRefreshToken();

  return result;
}

// Revoke All Tokens
export async function revokeAllTokens() {
  const currentToken = getRefreshToken();
  if (!currentToken) {
    return { success: false as const, error: 'No refresh token available' };
  }

  const result = await api.delete<RevokeTokenResponse>(
    '/account/public/revoke-all-tokens',
    { refresh_token: currentToken },
    { skipAuth: true }
  );

  // Clear local token
  clearRefreshToken();

  return result;
}

// Get Profile (Protected)
export async function getProfile() {
  return api.get<ProfileResponse>('/account/protected/profile');
}

// Generate Keys (Protected - Message Service)
export async function generateKeys() {
  return api.get<GenerateKeysResponse>('/message/protected/gen-keys');
}
