import { z } from 'zod';
import { LEASE_STATUSES, LEASE_TYPES } from '../types';

// Numeric and date fields are strings in the form (same as units'
// unitSchema.ts) and parsed at the API boundary — see leasesApi.ts's
// toCreateLeaseRequest. coResidents is a single comma-separated text
// field here rather than a real chip/array input, split into an array
// only at that same boundary — keeps this form as lightweight as
// UnitFormDialog rather than building a dedicated multi-value control
// for one field.
export const leaseSchema = z
  .object({
    leaseType: z.enum(LEASE_TYPES, { error: 'Select a lease type.' }),
    status: z.enum(LEASE_STATUSES),
    startDate: z.string().trim().min(1, 'Start date is required.'),
    endDate: z.string().trim().optional(),
    moveInDate: z.string().trim().optional(),
    moveOutDate: z.string().trim().optional(),
    monthlyRent: z.string().trim().min(1, 'Monthly rent is required.'),
    securityDeposit: z.string().trim().optional(),
    depositStatus: z.union([z.enum(['held', 'partially_returned', 'returned', 'forfeited']), z.literal('')]).optional(),
    rentDueDay: z.string().trim().optional(),
    lateFeeAmount: z.string().trim().optional(),
    lateFeeGraceDays: z.string().trim().optional(),
    primaryResidentName: z.string().trim().min(1, 'Primary resident name is required.'),
    coResidents: z.string().trim().optional(),
    emergencyContact: z.string().trim().optional(),
    renewalStatus: z.enum(['not_started', 'offered', 'accepted', 'declined']),
    terminationReason: z.union([z.enum(['non_renewal', 'eviction', 'mutual', 'resident_notice', 'other']), z.literal('')]).optional(),
    terminationNoticeDate: z.string().trim().optional(),
    signed: z.boolean(),
    signedDate: z.string().trim().optional(),
    notes: z.string().trim().optional(),
  })
  .refine((data) => data.leaseType !== 'fixed' || Boolean(data.endDate), {
    message: 'End date is required for a fixed-term lease.',
    path: ['endDate'],
  });

export type LeaseFormValues = z.infer<typeof leaseSchema>;

export function leaseDefaultValues(): LeaseFormValues {
  return {
    leaseType: undefined as unknown as LeaseFormValues['leaseType'],
    status: 'active',
    startDate: '',
    endDate: '',
    moveInDate: '',
    moveOutDate: '',
    monthlyRent: '',
    securityDeposit: '',
    depositStatus: '',
    rentDueDay: '',
    lateFeeAmount: '',
    lateFeeGraceDays: '',
    primaryResidentName: '',
    coResidents: '',
    emergencyContact: '',
    renewalStatus: 'not_started',
    terminationReason: '',
    terminationNoticeDate: '',
    signed: false,
    signedDate: '',
    notes: '',
  };
}
