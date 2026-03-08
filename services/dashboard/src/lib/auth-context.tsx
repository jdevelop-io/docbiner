'use client';

import { createContext, useContext, useEffect, useState, useCallback } from 'react';
import { api } from './api';
import type { User, Organization } from './types';

interface AuthState {
  user: User | null;
  organization: Organization | null;
  token: string | null;
  isLoading: boolean;
  isAuthenticated: boolean;
}

interface AuthContextValue extends AuthState {
  login: (token: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

const TOKEN_KEY = 'docbiner_token';

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>({
    user: null,
    organization: null,
    token: null,
    isLoading: true,
    isAuthenticated: false,
  });

  const fetchProfile = useCallback(async (token: string) => {
    api.setToken(token);
    try {
      const [user, organization] = await Promise.all([
        api.get<User>('/v1/auth/me'),
        api.get<Organization>('/v1/organization'),
      ]);
      setState({
        user,
        organization,
        token,
        isLoading: false,
        isAuthenticated: true,
      });
    } catch {
      localStorage.removeItem(TOKEN_KEY);
      api.clearToken();
      setState({
        user: null,
        organization: null,
        token: null,
        isLoading: false,
        isAuthenticated: false,
      });
    }
  }, []);

  useEffect(() => {
    const stored = localStorage.getItem(TOKEN_KEY);
    if (stored) {
      fetchProfile(stored);
    } else {
      setState((prev) => ({ ...prev, isLoading: false }));
    }
  }, [fetchProfile]);

  const login = useCallback(
    async (token: string) => {
      localStorage.setItem(TOKEN_KEY, token);
      await fetchProfile(token);
    },
    [fetchProfile],
  );

  const logout = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY);
    api.clearToken();
    setState({
      user: null,
      organization: null,
      token: null,
      isLoading: false,
      isAuthenticated: false,
    });
  }, []);

  return (
    <AuthContext.Provider value={{ ...state, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return ctx;
}
