import { z } from 'zod';
import { EMERGENCY_SLA_HOURS, WORK_ORDER_CATEGORIES, WORK_ORDER_PRIORITIES, WORK_ORDER_STATUSES } from '../types';

// Numeric/date fields are strings in the form (same as units/leases)
// and parsed at the API boundary — see workOrdersApi.ts's
// toWorkOrderRequest.
export const workOrderSchema = z
  .object({
    unitId: z.string().trim().optional(),
    title: z.string().trim().min(1, 'Title is required.'),
    description: z.string().trim().optional(),
    category: z.enum(WORK_ORDER_CATEGORIES, { error: 'Select a category.' }),
    priority: z.enum(WORK_ORDER_PRIORITIES),
    status: z.enum(WORK_ORDER_STATUSES),
    reportedBy: z.string().trim().optional(),
    reportedByContact: z.string().trim().optional(),
    assignedTo: z.string().trim().optional(),
    assignedToContact: z.string().trim().optional(),
    accessInstructions: z.string().trim().optional(),
    scheduledStart: z.string().trim().optional(),
    scheduledEnd: z.string().trim().optional(),
    dueDate: z.string().trim().optional(),
    estimatedCost: z.string().trim().optional(),
    actualCost: z.string().trim().optional(),
    photoLink: z.string().trim().optional(),
    invoiceLink: z.string().trim().optional(),
    internalNotes: z.string().trim().optional(),
  })
  .refine((data) => data.priority !== 'emergency' || Boolean(data.dueDate), {
    message: `A due date is required for emergency priority (within ${EMERGENCY_SLA_HOURS} hours).`,
    path: ['dueDate'],
  });

export type WorkOrderFormValues = z.infer<typeof workOrderSchema>;

export function workOrderDefaultValues(): WorkOrderFormValues {
  return {
    unitId: '',
    title: '',
    description: '',
    category: undefined as unknown as WorkOrderFormValues['category'],
    priority: 'medium',
    status: 'new',
    reportedBy: '',
    reportedByContact: '',
    assignedTo: '',
    assignedToContact: '',
    accessInstructions: '',
    scheduledStart: '',
    scheduledEnd: '',
    dueDate: '',
    estimatedCost: '',
    actualCost: '',
    photoLink: '',
    invoiceLink: '',
    internalNotes: '',
  };
}
