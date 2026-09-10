import { apiClient } from '@/api/client';
import type { AuthResponse, LoginRequest, RegisterRequest, User, UserRole } from '../types';

// Wire shapes exactly as the Go DTOs serialize them (snake_case). These
// stay private to this file; every other module in the app works with
// the camelCase types from ../types.
interface UserWire {
  id: string;
  email: string;
  role: UserRole;
  created_at: string;
}

interface AuthResponseWire {
  access_token: string;
  expires_in: number;
  user: UserWire;
}

function toUser(wire: UserWire): User {
  return { id: wire.id, email: wire.email, role: wire.role, createdAt: wire.created_at };
}

function toAuthResponse(wire: AuthResponseWire): AuthResponse {
  return { accessToken: wire.access_token, expiresIn: wire.expires_in, user: toUser(wire.user) };
}

export const authApi = {
  register: (payload: RegisterRequest): Promise<User> =>
    apiClient.post<UserWire>('/api/v1/auth/register', payload).then(toUser),

  login: (payload: LoginRequest): Promise<AuthResponse> =>
    apiClient.post<AuthResponseWire>('/api/v1/auth/login', payload).then(toAuthResponse),

  refresh: (): Promise<AuthResponse> =>
    apiClient
      .post<AuthResponseWire>('/api/v1/auth/refresh', undefined, { skipAuthRefresh: true })
      .then(toAuthResponse),

  logout: (): Promise<void> => apiClient.post<void>('/api/v1/auth/logout'),
};
