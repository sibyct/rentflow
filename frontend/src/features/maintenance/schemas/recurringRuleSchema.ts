import { z } from 'zod';
import { RECURRING_RULE_FREQUENCY_UNITS, WORK_ORDER_CATEGORIES } from '../types';

export const recurringRuleSchema = z.object({
  unitId: z.string().trim().optional(),
  title: z.string().trim().min(1, 'Title is required.'),
  description: z.string().trim().optional(),
  category: z.enum(WORK_ORDER_CATEGORIES, { error: 'Select a category.' }),
  frequencyInterval: z.string().trim().min(1, 'Enter a frequency.'),
  frequencyUnit: z.enum(RECURRING_RULE_FREQUENCY_UNITS),
  nextDueDate: z.string().trim().min(1, 'Next due date is required.'),
});

export type RecurringRuleFormValues = z.infer<typeof recurringRuleSchema>;

export function recurringRuleDefaultValues(): RecurringRuleFormValues {
  return {
    unitId: '',
    title: '',
    description: '',
    category: undefined as unknown as RecurringRuleFormValues['category'],
    frequencyInterval: '3',
    frequencyUnit: 'months',
    nextDueDate: '',
  };
}
