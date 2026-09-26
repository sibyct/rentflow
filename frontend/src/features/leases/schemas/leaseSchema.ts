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
    primaryResidentPhone: z.string().trim().optional(),
    primaryResidentEmail: z.string().trim().optional(),
    coResidents: z.string().trim().optional(),
    emergencyContact: z.string().trim().optional(),
    renewalStatus: z.enum(['not_started', 'offered', 'accepted', 'declined']),
    // Stored separately from monthlyRent/endDate — a proposal made while
    // Offered never touches the active lease's real terms (R9).
    proposedRent: z.string().trim().optional(),
    proposedEndDate: z.string().trim().optional(),
    offerSentDate: z.string().trim().optional(),
    terminationReason: z.union([z.enum(['non_renewal', 'eviction', 'mutual', 'resident_notice', 'other']), z.literal('')]).optional(),
    terminationNoticeDate: z.string().trim().optional(),
    signed: z.boolean(),
    signedDate: z.string().trim().optional(),
    notes: z.string().trim().optional(),
  })
  .refine((data) => data.leaseType !== 'fixed' || Boolean(data.endDate), {
    message: 'End date is required for a fixed-term lease.',
    path: ['endDate'],
  })
  // R8: entering "Offered" needs a proposed rent to show; entering
  // "Declined" needs a termination reason + notice date, same as an
  // actual termination.
  .refine((data) => data.renewalStatus !== 'offered' || Boolean(data.proposedRent), {
    message: 'Proposed rent is required when renewal status is offered.',
    path: ['proposedRent'],
  })
  .refine((data) => data.renewalStatus !== 'declined' || Boolean(data.terminationReason), {
    message: 'Termination reason is required when a renewal is declined.',
    path: ['terminationReason'],
  })
  .refine((data) => data.renewalStatus !== 'declined' || Boolean(data.terminationNoticeDate), {
    message: 'Termination notice date is required when a renewal is declined.',
    path: ['terminationNoticeDate'],
  })
  // R20: every monetary field is validated non-negative before save.
  // These mirror the backend's own `validate:"omitempty,min=0"` tags
  // (see dto/lease_dto.go) — without a client-side check too, a
  // negative value round-trips to the server and back as an
  // unhelpful top-level error instead of an inline field one.
  .refine((data) => !data.monthlyRent || Number(data.monthlyRent) >= 0, {
    message: 'Monthly rent cannot be negative.',
    path: ['monthlyRent'],
  })
  .refine((data) => !data.securityDeposit || Number(data.securityDeposit) >= 0, {
    message: 'Security deposit cannot be negative.',
    path: ['securityDeposit'],
  })
  .refine((data) => !data.lateFeeAmount || Number(data.lateFeeAmount) >= 0, {
    message: 'Late fee amount cannot be negative.',
    path: ['lateFeeAmount'],
  })
  .refine((data) => !data.lateFeeGraceDays || Number(data.lateFeeGraceDays) >= 0, {
    message: 'Late fee grace period cannot be negative.',
    path: ['lateFeeGraceDays'],
  })
  .refine((data) => !data.rentDueDay || (Number(data.rentDueDay) >= 1 && Number(data.rentDueDay) <= 31), {
    message: 'Rent due day must be between 1 and 31.',
    path: ['rentDueDay'],
  })
  .refine((data) => !data.proposedRent || Number(data.proposedRent) >= 0, {
    message: 'Proposed rent cannot be negative.',
    path: ['proposedRent'],
  });

export type LeaseFormValues = z.infer<typeof leaseSchema>;

export function leaseDefaultValues(): LeaseFormValues {
  return {
    leaseType: undefined as unknown as LeaseFormValues['leaseType'],
    // Draft is the real baseline for a brand-new lease — LeaseFormDialog
    // has no Status field on create at all, and computes the actual
    // submitted status itself (draft, or active for a signed
    // historical-entry import) rather than trusting this default.
    status: 'draft',
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
    primaryResidentPhone: '',
    primaryResidentEmail: '',
    coResidents: '',
    emergencyContact: '',
    renewalStatus: 'not_started',
    proposedRent: '',
    proposedEndDate: '',
    offerSentDate: '',
    terminationReason: '',
    terminationNoticeDate: '',
    signed: false,
    signedDate: '',
    notes: '',
  };
}
