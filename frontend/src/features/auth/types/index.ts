// Mirrors backend/internal/transport/http/dto/auth_dto.go

export type UserRole = 'admin' | 'manager' | 'tenant';

export interface User {
  id: string;
  email: string;
  role: UserRole;
  createdAt: string;
}

export interface AuthResponse {
  accessToken: string;
  expiresIn: number;
  user: User;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
}
