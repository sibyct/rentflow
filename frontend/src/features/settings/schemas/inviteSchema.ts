import { z } from 'zod';
import { STAFF_ROLES } from '../types';

export const inviteSchema = z
  .object({
    name: z.string().trim().min(1, 'Name is required.').max(120),
    email: z.string().trim().min(1, 'Email is required.').email('Enter a valid email address.'),
    role: z.enum(STAFF_ROLES, { error: 'Choose a role.' }),
    accessAll: z.boolean(),
    properties: z.array(z.string()),
  })
  .refine((v) => v.role === 'admin' || v.accessAll || v.properties.length > 0, {
    path: ['properties'],
    message: 'Choose at least one property, or grant access to all.',
  });

export type InviteFormValues = z.infer<typeof inviteSchema>;

export function inviteDefaultValues(): InviteFormValues {
  return { name: '', email: '', role: undefined as unknown as InviteFormValues['role'], accessAll: true, properties: [] };
}
