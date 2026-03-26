'use client';

import React, {
  createContext,
  useContext,
  useReducer,
  useEffect,
  useCallback,
} from 'react';
import { apiClient, setAccessToken } from '@/lib/api-client';
import type { User, LoginInput, RegisterInput } from '@/types/api';

// ─── State ───────────────────────────────────────────────────────────────────

interface AuthState {
  user: User | null;
  status: 'loading' | 'authenticated' | 'unauthenticated';
}

type AuthAction =
  | { type: 'SET_USER'; payload: User; token: string }
  | { type: 'LOGOUT' }
  | { type: 'SET_LOADING' }
  | { type: 'SET_UNAUTHENTICATED' };

function authReducer(state: AuthState, action: AuthAction): AuthState {
  switch (action.type) {
    case 'SET_USER':
      return { user: action.payload, status: 'authenticated' };
    case 'LOGOUT':
      return { user: null, status: 'unauthenticated' };
    case 'SET_LOADING':
      return { ...state, status: 'loading' };
    case 'SET_UNAUTHENTICATED':
      return { user: null, status: 'unauthenticated' };
  }
}

// ─── Context ──────────────────────────────────────────────────────────────────

interface AuthContextValue {
  user: User | null;
  status: 'loading' | 'authenticated' | 'unauthenticated';
  login: (data: LoginInput) => Promise<void>;
  register: (data: RegisterInput) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

// ─── Provider ────────────────────────────────────────────────────────────────

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, dispatch] = useReducer(authReducer, {
    user: null,
    status: 'loading',
  });

  // On mount, attempt silent refresh to restore session
  useEffect(() => {
    apiClient.auth
      .refresh()
      .then(async (res) => {
        setAccessToken(res.data.access_token);
        const userRes = await apiClient.users.me();
        dispatch({ type: 'SET_USER', payload: userRes.data, token: res.data.access_token });
      })
      .catch(() => {
        dispatch({ type: 'SET_UNAUTHENTICATED' });
      });
  }, []);

  const login = useCallback(async (data: LoginInput) => {
    const res = await apiClient.auth.login(data);
    setAccessToken(res.data.access_token);
    dispatch({ type: 'SET_USER', payload: res.data.user, token: res.data.access_token });
  }, []);

  const register = useCallback(async (data: RegisterInput) => {
    const res = await apiClient.auth.register(data);
    setAccessToken(res.data.access_token);
    dispatch({ type: 'SET_USER', payload: res.data.user, token: res.data.access_token });
  }, []);

  const logout = useCallback(async () => {
    try {
      await apiClient.auth.logout();
    } catch {
      // ignore
    }
    setAccessToken(null);
    dispatch({ type: 'LOGOUT' });
  }, []);

  return (
    <AuthContext.Provider value={{ ...state, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider');
  return ctx;
}
