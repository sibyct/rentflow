import { z } from 'zod';
import { dollarsToCents, todayIso } from '@/shared/lib/format';
import { CHARGE_TYPES } from '../types';

export const chargeSchema = z.object({
  unitId: z.string().min(1, 'Choose a unit.'),
  chargeType: z.enum(CHARGE_TYPES, { error: 'Choose a charge type.' }),
  amount: z
    .string()
    .trim()
    .min(1, 'Amount is required.')
    .refine((v) => (dollarsToCents(v) ?? 0) > 0, 'Enter an amount greater than zero.'),
  description: z.string().trim().min(1, 'Add a short description.').max(500),
  dueOn: z.string().min(1, 'Due date is required.'),
});

export type ChargeFormValues = z.infer<typeof chargeSchema>;

export function chargeDefaultValues(): ChargeFormValues {
  return {
    unitId: '',
    chargeType: undefined as unknown as ChargeFormValues['chargeType'],
    amount: '',
    description: '',
    dueOn: todayIso(),
  };
}
