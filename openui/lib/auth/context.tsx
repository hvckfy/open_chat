'use client';

import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from 'react';
import type { User, LoginRequest, RegisterRequest } from '@/lib/api/types';
import {
  loginLdap,
  loginLocal,
  registerLocal,
  revokeToken,
  getProfile,
} from '@/lib/api/auth';
import { hasRefreshToken, clearRefreshToken } from '@/lib/api/client';

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (credentials: LoginRequest, method: 'ldap' | 'local') => Promise<{ success: boolean; error?: string }>;
  register: (userData: RegisterRequest) => Promise<{ success: boolean; error?: string }>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Check auth status on mount
  useEffect(() => {
    checkAuthStatus();
  }, []);

  const checkAuthStatus = async () => {
    setIsLoading(true);
    
    // If no refresh token, user is not authenticated
    if (!hasRefreshToken()) {
      setUser(null);
      setIsLoading(false);
      return;
    }

    // Try to fetch profile to validate authentication
    const result = await getProfile();
    if (result.success) {
      setUser(result.data);
    } else {
      // Token invalid, clear it
      clearRefreshToken();
      setUser(null);
    }
    
    setIsLoading(false);
  };

  const login = useCallback(
    async (credentials: LoginRequest, method: 'ldap' | 'local') => {
      setIsLoading(true);
      
      const loginFn = method === 'ldap' ? loginLdap : loginLocal;
      const result = await loginFn(credentials);

      if (result.success) {
        // Fetch user profile after successful login
        const profileResult = await getProfile();
        if (profileResult.success) {
          setUser(profileResult.data);
          setIsLoading(false);
          return { success: true };
        } else {
          setIsLoading(false);
          return { success: false, error: 'Failed to fetch user profile' };
        }
      }

      setIsLoading(false);
      return { success: false, error: result.error };
    },
    []
  );

  const register = useCallback(async (userData: RegisterRequest) => {
    setIsLoading(true);
    
    const result = await registerLocal(userData);

    if (result.success) {
      // Fetch user profile after successful registration
      const profileResult = await getProfile();
      if (profileResult.success) {
        setUser(profileResult.data);
        setIsLoading(false);
        return { success: true };
      } else {
        setIsLoading(false);
        return { success: false, error: 'Failed to fetch user profile' };
      }
    }

    setIsLoading(false);
    return { success: false, error: result.error };
  }, []);

  const logout = useCallback(async () => {
    setIsLoading(true);
    await revokeToken();
    setUser(null);
    setIsLoading(false);
  }, []);

  const refreshUser = useCallback(async () => {
    const result = await getProfile();
    if (result.success) {
      setUser(result.data);
    }
  }, []);

  const value: AuthContextType = {
    user,
    isAuthenticated: !!user,
    isLoading,
    login,
    register,
    logout,
    refreshUser,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
