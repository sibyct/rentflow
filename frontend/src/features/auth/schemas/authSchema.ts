import { z } from 'zod';

// Mirrors the `validate` tags on backend/internal/transport/http/dto/auth_dto.go
// (RegisterRequest/LoginRequest) — client-side validation is a UX
// nicety, not a substitute for the server's own checks, but keeping the
// rules in sync means the client rarely round-trips just to be told
// "email is required."

export const loginSchema = z.object({
  email: z.string().trim().min(1, 'Email is required').email('Must be a valid email address'),
  password: z.string().min(1, 'Password is required'),
});

export type LoginFormValues = z.infer<typeof loginSchema>;

export const registerSchema = z.object({
  email: z.string().trim().min(1, 'Email is required').email('Must be a valid email address'),
  password: z
    .string()
    .min(8, 'Password must be at least 8 characters')
    .max(72, 'Password must be at most 72 characters'),
});

export type RegisterFormValues = z.infer<typeof registerSchema>;
