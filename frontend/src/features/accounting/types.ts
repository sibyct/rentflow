import type { StatusTone } from '@/shared/components';

// Money is integer cents everywhere in this feature; see shared/lib/format.

export type PaymentStatus = 'paid' | 'partial' | 'late' | 'unpaid';
export const PAYMENT_STATUSES: PaymentStatus[] = ['paid', 'partial', 'late', 'unpaid'];
export const PAYMENT_STATUS_LABELS: Record<PaymentStatus, string> = {
  paid: 'Paid',
  partial: 'Partial',
  late: 'Late',
  unpaid: 'Unpaid',
};
export const PAYMENT_STATUS_TONE: Record<PaymentStatus, StatusTone> = {
  paid: 'success',
  partial: 'warning',
  late: 'error',
  unpaid: 'default',
};

export type ExpenseStatus = 'unpaid' | 'paid' | 'overdue';
export const EXPENSE_STATUSES: ExpenseStatus[] = ['unpaid', 'paid', 'overdue'];
export const EXPENSE_STATUS_LABELS: Record<ExpenseStatus, string> = {
  unpaid: 'Unpaid',
  paid: 'Paid',
  overdue: 'Overdue',
};
export const EXPENSE_STATUS_TONE: Record<ExpenseStatus, StatusTone> = {
  unpaid: 'warning',
  paid: 'success',
  overdue: 'error',
};

export type ExpenseCategory =
  | 'repairs'
  | 'utilities'
  | 'insurance'
  | 'property_tax'
  | 'management_fee'
  | 'landscaping'
  | 'supplies'
  | 'legal'
  | 'mortgage'
  | 'other';
export const EXPENSE_CATEGORIES: ExpenseCategory[] = [
  'repairs',
  'utilities',
  'insurance',
  'property_tax',
  'management_fee',
  'landscaping',
  'supplies',
  'legal',
  'mortgage',
  'other',
];
export const EXPENSE_CATEGORY_LABELS: Record<ExpenseCategory, string> = {
  repairs: 'Repairs',
  utilities: 'Utilities',
  insurance: 'Insurance',
  property_tax: 'Property Tax',
  management_fee: 'Management Fee',
  landscaping: 'Landscaping',
  supplies: 'Supplies',
  legal: 'Legal',
  mortgage: 'Mortgage',
  other: 'Other',
};

export type ChargeType = 'utility_rebill' | 'damage' | 'amenity' | 'other';
export const CHARGE_TYPES: ChargeType[] = ['utility_rebill', 'damage', 'amenity', 'other'];
export const CHARGE_TYPE_LABELS: Record<ChargeType, string> = {
  utility_rebill: 'Utility Rebill',
  damage: 'Damage Charge',
  amenity: 'Amenity Fee',
  other: 'Other',
};

export type RecurrenceFrequency = 'monthly' | 'quarterly' | 'yearly';
export const RECURRENCE_FREQUENCIES: RecurrenceFrequency[] = ['monthly', 'quarterly', 'yearly'];
export const RECURRENCE_LABELS: Record<RecurrenceFrequency, string> = {
  monthly: 'Monthly',
  quarterly: 'Quarterly',
  yearly: 'Yearly',
};

export const PAYMENT_METHODS = ['ACH', 'Check', 'Cash', 'Card', 'Wire', 'Other'];

export type DashboardPeriod = 'month' | '6months' | 'ytd';

export interface RentRollRow {
  leaseId: string;
  propertyId: string;
  propertyName: string;
  unitId: string;
  unitName: string;
  tenantName: string;
  /** First of the billing month, "YYYY-MM-DD". */
  period: string;
  rentCents: number;
  dueOn: string;
  billedCents: number;
  paidCents: number;
  outstandingCents: number;
  status: PaymentStatus;
  lastPaidOn: string;
  graceDays: number;
}

/** An expense or a one-off charge — both are ledger rows. */
export interface LedgerRow {
  id: string;
  propertyId: string;
  propertyName: string;
  unitId: string;
  unitName: string;
  tenantName: string;
  leaseId: string;
  type: 'rent' | 'late_fee' | 'charge' | 'expense';
  chargeType: ChargeType | '';
  category: ExpenseCategory | '';
  amountCents: number;
  paidCents: number;
  outstandingCents: number;
  /** ExpenseStatus for an expense, PaymentStatus for a charge. */
  status: string;
  incurredOn: string;
  dueOn: string;
  lastPaidOn: string;
  vendorId: string;
  vendorName: string;
  workOrderId: string;
  workOrderTitle: string;
  description: string;
  taxDeductible: boolean;
  isRecurring: boolean;
  recurrenceFrequency: RecurrenceFrequency | '';
  attachmentId: string;
  source: string;
  /** Created from a work order — edited there, not here. */
  readOnly: boolean;
  createdAt: string;
}

export interface Payment {
  id: string;
  transactionId: string;
  amountCents: number;
  paidOn: string;
  method: string;
  reference: string;
  bankAccountId: string;
  voided: boolean;
  createdAt: string;
}

export interface AuditEntry {
  id: string;
  action: string;
  actorId: string;
  changes: Record<string, { old: unknown; new: unknown }>;
  createdAt: string;
}

export interface AccountingSettings {
  lateFeeKind: 'flat' | 'percent';
  /** Cents when flat, basis points (100 = 1%) when percent. */
  lateFeeValue: number;
  graceDays: number;
  defaultRentDueDay: number;
}

export interface SeriesPoint {
  bucket: string;
  incomeCents: number;
  expenseCents: number;
}

export interface AccountingDashboard {
  period: DashboardPeriod;
  collectedCents: number;
  expectedCents: number;
  outstandingCents: number;
  expensesPaidCents: number;
  netIncomeCents: number;
  upcomingExpenseCents: number;
  upcomingExpenseCount: number;
  overdueExpenseCount: number;
  lateRentCount: number;
  series: SeriesPoint[];
}
