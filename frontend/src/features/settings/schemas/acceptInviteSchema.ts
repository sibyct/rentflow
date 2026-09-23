import { z } from 'zod';

// Mirrors registerSchema's password rule (features/auth/schemas/authSchema.ts)
// — this is a front-end-only prototype (see ../types.ts), so there is no
// backend endpoint this actually posts to.
export const acceptInviteSchema = z
  .object({
    password: z.string().min(8, 'Password must be at least 8 characters').max(72, 'Password must be at most 72 characters'),
    confirmPassword: z.string().min(1, 'Confirm your password'),
  })
  .refine((v) => v.password === v.confirmPassword, { path: ['confirmPassword'], message: "Passwords don't match." });

export type AcceptInviteFormValues = z.infer<typeof acceptInviteSchema>;
