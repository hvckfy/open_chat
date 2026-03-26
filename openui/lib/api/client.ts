import type { ApiResponse, ApiError } from './types';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';

// Token storage keys
const REFRESH_TOKEN_KEY = 'openchat_refresh_token';

// Get refresh token from localStorage
export function getRefreshToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

// Set refresh token in localStorage
export function setRefreshToken(token: string): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem(REFRESH_TOKEN_KEY, token);
}

// Clear refresh token from localStorage
export function clearRefreshToken(): void {
  if (typeof window === 'undefined') return;
  localStorage.removeItem(REFRESH_TOKEN_KEY);
}

// Check if we have a refresh token (basic auth check)
export function hasRefreshToken(): boolean {
  return !!getRefreshToken();
}

interface FetchOptions extends RequestInit {
  skipAuth?: boolean;
}

// Attempt to refresh the access token
async function refreshAccessToken(): Promise<boolean> {
  const refreshToken = getRefreshToken();
  if (!refreshToken) return false;

  try {
    const response = await fetch(`${API_BASE_URL}/account/public/refresh-token`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!response.ok) {
      clearRefreshToken();
      return false;
    }

    return true;
  } catch {
    clearRefreshToken();
    return false;
  }
}

// Main fetch wrapper with automatic token refresh
export async function apiRequest<T>(
  endpoint: string,
  options: FetchOptions = {}
): Promise<ApiResponse<T>> {
  const { skipAuth = false, ...fetchOptions } = options;

  const url = endpoint.startsWith('http') ? endpoint : `${API_BASE_URL}${endpoint}`;

  const defaultOptions: RequestInit = {
    credentials: 'include', // Always include cookies for access_token
    headers: {
      'Content-Type': 'application/json',
      ...fetchOptions.headers,
    },
  };

  const mergedOptions = { ...defaultOptions, ...fetchOptions };

  try {
    let response = await fetch(url, mergedOptions);

    // If 401 and we have a refresh token, try to refresh
    if (response.status === 401 && !skipAuth && hasRefreshToken()) {
      const refreshed = await refreshAccessToken();
      if (refreshed) {
        // Retry the original request
        response = await fetch(url, mergedOptions);
      }
    }

    // Handle non-ok responses
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Unknown error' })) as ApiError;
      return {
        success: false,
        error: errorData.error || `HTTP ${response.status}: ${response.statusText}`,
      };
    }

    // Parse successful response
    const data = await response.json() as T;
    return { success: true, data };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Network error',
    };
  }
}

// Convenience methods
export const api = {
  get: <T>(endpoint: string, options?: FetchOptions) =>
    apiRequest<T>(endpoint, { ...options, method: 'GET' }),

  post: <T>(endpoint: string, body?: unknown, options?: FetchOptions) =>
    apiRequest<T>(endpoint, {
      ...options,
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    }),

  put: <T>(endpoint: string, body?: unknown, options?: FetchOptions) =>
    apiRequest<T>(endpoint, {
      ...options,
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    }),

  delete: <T>(endpoint: string, body?: unknown, options?: FetchOptions) =>
    apiRequest<T>(endpoint, {
      ...options,
      method: 'DELETE',
      body: body ? JSON.stringify(body) : undefined,
    }),
};

export { API_BASE_URL };
