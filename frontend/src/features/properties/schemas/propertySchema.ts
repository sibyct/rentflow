import { z } from 'zod';

// Mirrors the `validate` tags on
// backend/internal/transport/http/dto/property_dto.go (CreatePropertyRequest)
// — same rationale as features/auth/schemas/authSchema.ts.

export const propertyStatusSchema = z.enum(['active', 'inactive', 'maintenance']);

export const createPropertySchema = z.object({
  address: z
    .string()
    .trim()
    .min(3, 'Address must be at least 3 characters')
    .max(255, 'Address must be at most 255 characters'),
  unitCount: z
    .number({ error: 'Unit count must be a number' })
    .int('Unit count must be a whole number')
    .min(1, 'Unit count must be at least 1')
    .max(100000, 'Unit count must be at most 100000'),
  status: propertyStatusSchema,
});

export type CreatePropertyFormValues = z.infer<typeof createPropertySchema>;
