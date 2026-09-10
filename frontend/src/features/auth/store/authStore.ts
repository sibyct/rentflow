import { create } from 'zustand';
import type { User } from '../types';

interface AuthState {
  accessToken: string | null;
  user: User | null;
  isAuthenticated: boolean;
  setAuth: (accessToken: string, user: User) => void;
  setAccessToken: (accessToken: string) => void;
  clear: () => void;
}

// The access token lives only in memory (this store), never in
// localStorage — the refresh token that can mint new ones is an
// httpOnly cookie the client never touches directly. Losing this store
// on a hard refresh is intentional: src/app/providers.tsx calls
// /auth/refresh once on boot to silently re-establish the session from
// that cookie.
export const useAuthStore = create<AuthState>((set) => ({
  accessToken: null,
  user: null,
  isAuthenticated: false,
  setAuth: (accessToken, user) => set({ accessToken, user, isAuthenticated: true }),
  setAccessToken: (accessToken) => set({ accessToken, isAuthenticated: true }),
  clear: () => set({ accessToken: null, user: null, isAuthenticated: false }),
}));
