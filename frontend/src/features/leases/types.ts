// Display types for the Leases feature — a time-bound contract binding
// a unit to whoever is occupying it. Mirrors features/units/types.ts's
// role: no mock data, this is real from day one.

export type LeaseType = 'fixed' | 'month_to_month';

export const LEASE_TYPES: LeaseType[] = ['fixed', 'month_to_month'];

export const LEASE_TYPE_LABELS: Record<LeaseType, string> = {
  fixed: 'Fixed term',
  month_to_month: 'Month-to-month',
};

// The three states a manager actually sets by taking an action — see
// LeaseDisplayStatus for the richer, date-derived status a list/detail
// view shows instead.
export type LeaseStatus = 'draft' | 'active' | 'terminated';

export const LEASE_STATUSES: LeaseStatus[] = ['draft', 'active', 'terminated'];

// What a lease list/detail view actually shows — 'active' fans out into
// upcoming/active/expiring_soon/expired depending on today's date, the
// same computation the backend does in domain.Lease.DisplayStatus; the
// frontend never recomputes this, it just reads display_status off the
// wire.
export type LeaseDisplayStatus = 'draft' | 'upcoming' | 'active' | 'expiring_soon' | 'expired' | 'terminated';

export const LEASE_DISPLAY_STATUSES: LeaseDisplayStatus[] = ['draft', 'upcoming', 'active', 'expiring_soon', 'expired', 'terminated'];

export const LEASE_DISPLAY_STATUS_LABELS: Record<LeaseDisplayStatus, string> = {
  draft: 'Draft',
  upcoming: 'Upcoming',
  active: 'Active',
  expiring_soon: 'Expiring soon',
  expired: 'Expired',
  terminated: 'Terminated',
};

export type DepositStatus = 'held' | 'partially_returned' | 'returned' | 'forfeited' | '';

export const DEPOSIT_STATUS_OPTIONS: Exclude<DepositStatus, ''>[] = ['held', 'partially_returned', 'returned', 'forfeited'];

export const DEPOSIT_STATUS_LABELS: Record<Exclude<DepositStatus, ''>, string> = {
  held: 'Held',
  partially_returned: 'Partially returned',
  returned: 'Returned',
  forfeited: 'Forfeited',
};

export type RenewalStatus = 'not_started' | 'offered' | 'accepted' | 'declined';

export const RENEWAL_STATUSES: RenewalStatus[] = ['not_started', 'offered', 'accepted', 'declined'];

export const RENEWAL_STATUS_LABELS: Record<RenewalStatus, string> = {
  not_started: 'Not started',
  offered: 'Offered',
  accepted: 'Accepted',
  declined: 'Declined',
};

export type TerminationReason = 'non_renewal' | 'eviction' | 'mutual' | 'resident_notice' | 'other' | '';

export const TERMINATION_REASON_OPTIONS: Exclude<TerminationReason, ''>[] = ['non_renewal', 'eviction', 'mutual', 'resident_notice', 'other'];

export const TERMINATION_REASON_LABELS: Record<Exclude<TerminationReason, ''>, string> = {
  non_renewal: 'Non-renewal',
  eviction: 'Eviction',
  mutual: 'Mutual agreement',
  resident_notice: "Resident's notice",
  other: 'Other',
};

export interface LeaseRow {
  id: string;
  unitId: string;
  leaseType: LeaseType;
  status: LeaseStatus;
  displayStatus: LeaseDisplayStatus;
  startDate: string;
  endDate: string;
  monthlyRent: number;
  primaryResidentName: string;
  coResidents: string[];
  renewalStatus: RenewalStatus;
  signed: boolean;
}

/** Full record for the lease detail/edit views — everything LeaseRow has, plus every field the Add/Edit Lease form collects. */
export interface LeaseDetail extends LeaseRow {
  moveInDate: string;
  moveOutDate: string;
  securityDeposit: number | null;
  depositStatus: DepositStatus;
  rentDueDay: number | null;
  lateFeeAmount: number | null;
  lateFeeGraceDays: number | null;
  emergencyContact: string;
  terminationReason: TerminationReason;
  terminationNoticeDate: string;
  signedDate: string;
  notes: string;
  createdAt: string;
  updatedAt: string;
}

/** A LeaseRow decorated with its unit's and parent property's names — the shape the portfolio-wide global Leases page (and a lease detail view's breadcrumb) needs. */
export interface LeaseRowWithUnitProperty extends LeaseRow {
  unitName: string;
  propertyId: string;
  propertyName: string;
}

export interface LeaseDetailWithUnitProperty extends LeaseDetail {
  unitName: string;
  propertyId: string;
  propertyName: string;
}
