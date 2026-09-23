import { z } from 'zod';
import { dollarsToCents, todayIso } from '@/shared/lib/format';
import { EXPENSE_CATEGORIES, RECURRENCE_FREQUENCIES } from '../types';

// Form values stay strings (dollars, ISO dates) exactly as the inputs hold
// them; accountingApi converts to cents / omits blanks at the boundary.
export const expenseSchema = z
  .object({
    propertyId: z.string().min(1, 'Choose a property.'),
    unitId: z.string(),
    category: z.enum(EXPENSE_CATEGORIES, { error: 'Choose a category.' }),
    vendorId: z.string(),
    vendorName: z.string().trim().max(200),
    amount: z
      .string()
      .trim()
      .min(1, 'Amount is required.')
      .refine((v) => (dollarsToCents(v) ?? 0) > 0, 'Enter an amount greater than zero.'),
    incurredOn: z.string().min(1, 'Date incurred is required.'),
    dueOn: z.string(),
    description: z.string().trim().max(2000),
    taxDeductible: z.boolean(),
    isRecurring: z.boolean(),
    recurrenceFrequency: z.union([z.enum(RECURRENCE_FREQUENCIES), z.literal('')]),
    paidOn: z.string(),
    paymentMethod: z.string(),
    attachmentId: z.string(),
  })
  .refine((v) => !v.isRecurring || v.recurrenceFrequency !== '', {
    path: ['recurrenceFrequency'],
    message: 'Choose how often it repeats.',
  })
  .refine((v) => !v.dueOn || v.dueOn >= v.incurredOn, {
    path: ['dueOn'],
    message: 'Due date cannot be before the date incurred.',
  });

export type ExpenseFormValues = z.infer<typeof expenseSchema>;

export function expenseDefaultValues(propertyId = '', unitId = ''): ExpenseFormValues {
  return {
    propertyId,
    unitId,
    category: undefined as unknown as ExpenseFormValues['category'],
    vendorId: '',
    vendorName: '',
    amount: '',
    incurredOn: todayIso(),
    dueOn: '',
    description: '',
    taxDeductible: false,
    isRecurring: false,
    recurrenceFrequency: '',
    paidOn: '',
    paymentMethod: '',
    attachmentId: '',
  };
}
