// API Types based on api.md specification

// User types
export interface UserData {
  firstName: string;
  secondName: string;
}

export interface UserPersonal {
  mail: string;
  phone: string;
}

export interface UserApp {
  userId: number;
  username: string;
  authType: 'ldap' | 'local';
}

export interface User {
  data: UserData;
  personal: UserPersonal;
  app: UserApp;
}

// Auth request/response types
export interface LoginRequest {
  username: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  password: string;
  firstName: string;
  secondName: string;
  mail: string;
  phone: string;
}

export interface AuthResponse {
  refresh_token: string;
  expires_in: number;
}

export interface RefreshTokenRequest {
  refresh_token: string;
}

export interface RefreshTokenResponse {
  expires_in: number;
}

export interface RevokeTokenRequest {
  refresh_token: string;
}

export interface RevokeTokenResponse {
  message: boolean;
}

// Profile response
export interface ProfileResponse extends User {}

// Message service types
export interface GenerateKeysResponse {
  words: string[];
  private_key: string;
}

// Health check
export interface HealthResponse {
  status: 'ok';
}

// Error response
export interface ApiError {
  error: string;
}

// Generic API response wrapper
export type ApiResponse<T> = 
  | { success: true; data: T }
  | { success: false; error: string };
