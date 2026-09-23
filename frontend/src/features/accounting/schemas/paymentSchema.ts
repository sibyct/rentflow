import { z } from 'zod';
import { centsToDollarsInput, dollarsToCents, todayIso } from '@/shared/lib/format';

export const paymentSchema = z.object({
  amount: z
    .string()
    .trim()
    .min(1, 'Amount is required.')
    .refine((v) => (dollarsToCents(v) ?? 0) > 0, 'Enter an amount greater than zero.'),
  paidOn: z.string().min(1, 'Payment date is required.'),
  method: z.string(),
  reference: z.string().trim().max(120),
});

export type PaymentFormValues = z.infer<typeof paymentSchema>;

/** Pre-fills the amount with what is still owed — the common case is paying it off. */
export function paymentDefaultValues(outstandingCents = 0): PaymentFormValues {
  return {
    amount: outstandingCents > 0 ? centsToDollarsInput(outstandingCents) : '',
    paidOn: todayIso(),
    method: '',
    reference: '',
  };
}
