import { apiClient } from '@/api/client';
import { dollarsToCents } from '@/shared/lib/format';
import type {
  AccountingDashboard,
  AccountingSettings,
  AuditEntry,
  ChargeType,
  DashboardPeriod,
  ExpenseCategory,
  ExpenseStatus,
  LedgerRow,
  Payment,
  PaymentStatus,
  RecurrenceFrequency,
  RentRollRow,
  SeriesPoint,
} from '../types';
import type { ExpenseFormValues } from '../schemas/expenseSchema';
import type { ChargeFormValues } from '../schemas/chargeSchema';
import type { PaymentFormValues } from '../schemas/paymentSchema';

// Wire shapes mirror backend/internal/transport/http/dto/accounting_dto.go
// (snake_case). They stay private to this file; everything else works
// with the camelCase types in ../types.

interface ListMeta {
  total: number;
  limit: number;
  offset: number;
}

interface LedgerWire {
  id: string;
  property_id: string;
  property_name: string;
  unit_id?: string;
  unit_name?: string;
  tenant_name?: string;
  lease_id?: string;
  type: LedgerRow['type'];
  charge_type?: string;
  category?: string;
  amount_cents: number;
  paid_cents: number;
  outstanding_cents: number;
  status: string;
  incurred_on: string;
  due_on?: string;
  last_paid_on?: string;
  vendor_id?: string;
  vendor_name?: string;
  work_order_id?: string;
  work_order_title?: string;
  description?: string;
  tax_deductible: boolean;
  is_recurring: boolean;
  recurrence_frequency?: string;
  attachment_id?: string;
  source: string;
  read_only: boolean;
  created_at: string;
}

function toLedgerRow(w: LedgerWire): LedgerRow {
  return {
    id: w.id,
    propertyId: w.property_id,
    propertyName: w.property_name,
    unitId: w.unit_id ?? '',
    unitName: w.unit_name ?? '',
    tenantName: w.tenant_name ?? '',
    leaseId: w.lease_id ?? '',
    type: w.type,
    chargeType: (w.charge_type ?? '') as ChargeType | '',
    category: (w.category ?? '') as ExpenseCategory | '',
    amountCents: w.amount_cents,
    paidCents: w.paid_cents,
    outstandingCents: w.outstanding_cents,
    status: w.status,
    incurredOn: w.incurred_on,
    dueOn: w.due_on ?? '',
    lastPaidOn: w.last_paid_on ?? '',
    vendorId: w.vendor_id ?? '',
    vendorName: w.vendor_name ?? '',
    workOrderId: w.work_order_id ?? '',
    workOrderTitle: w.work_order_title ?? '',
    description: w.description ?? '',
    taxDeductible: w.tax_deductible,
    isRecurring: w.is_recurring,
    recurrenceFrequency: (w.recurrence_frequency ?? '') as RecurrenceFrequency | '',
    attachmentId: w.attachment_id ?? '',
    source: w.source,
    readOnly: w.read_only,
    createdAt: w.created_at,
  };
}

interface RentRollWire {
  lease_id: string;
  property_id: string;
  property_name: string;
  unit_id: string;
  unit_name: string;
  tenant_name: string;
  period: string;
  rent_cents: number;
  due_on: string;
  billed_cents: number;
  paid_cents: number;
  outstanding_cents: number;
  status: PaymentStatus;
  last_paid_on?: string;
  grace_days: number;
}

function toRentRollRow(w: RentRollWire): RentRollRow {
  return {
    leaseId: w.lease_id,
    propertyId: w.property_id,
    propertyName: w.property_name,
    unitId: w.unit_id,
    unitName: w.unit_name,
    tenantName: w.tenant_name,
    period: w.period,
    rentCents: w.rent_cents,
    dueOn: w.due_on,
    billedCents: w.billed_cents,
    paidCents: w.paid_cents,
    outstandingCents: w.outstanding_cents,
    status: w.status,
    lastPaidOn: w.last_paid_on ?? '',
    graceDays: w.grace_days,
  };
}

interface PaymentWire {
  id: string;
  transaction_id: string;
  amount_cents: number;
  paid_on: string;
  method?: string;
  reference?: string;
  bank_account_id?: string;
  voided: boolean;
  created_at: string;
}

function toPayment(w: PaymentWire): Payment {
  return {
    id: w.id,
    transactionId: w.transaction_id,
    amountCents: w.amount_cents,
    paidOn: w.paid_on,
    method: w.method ?? '',
    reference: w.reference ?? '',
    bankAccountId: w.bank_account_id ?? '',
    voided: w.voided,
    createdAt: w.created_at,
  };
}

interface SettingsWire {
  late_fee_kind: 'flat' | 'percent';
  late_fee_value: number;
  grace_days: number;
  default_rent_due_day: number;
}

function toSettings(w: SettingsWire): AccountingSettings {
  return {
    lateFeeKind: w.late_fee_kind,
    lateFeeValue: w.late_fee_value,
    graceDays: w.grace_days,
    defaultRentDueDay: w.default_rent_due_day,
  };
}

interface DashboardWire {
  period: DashboardPeriod;
  collected_cents: number;
  expected_cents: number;
  outstanding_cents: number;
  expenses_paid_cents: number;
  net_income_cents: number;
  upcoming_expense_cents: number;
  upcoming_expense_count: number;
  overdue_expense_count: number;
  late_rent_count: number;
  series: { bucket: string; income_cents: number; expense_cents: number }[];
}

function toDashboard(w: DashboardWire): AccountingDashboard {
  const series: SeriesPoint[] = w.series.map((p) => ({
    bucket: p.bucket,
    incomeCents: p.income_cents,
    expenseCents: p.expense_cents,
  }));
  return {
    period: w.period,
    collectedCents: w.collected_cents,
    expectedCents: w.expected_cents,
    outstandingCents: w.outstanding_cents,
    expensesPaidCents: w.expenses_paid_cents,
    netIncomeCents: w.net_income_cents,
    upcomingExpenseCents: w.upcoming_expense_cents,
    upcomingExpenseCount: w.upcoming_expense_count,
    overdueExpenseCount: w.overdue_expense_count,
    lateRentCount: w.late_rent_count,
    series,
  };
}

// ---- Form values → request bodies ----

export function toExpenseRequest(values: ExpenseFormValues, mode: 'create' | 'update') {
  const body: Record<string, unknown> = {
    property_id: values.propertyId,
    unit_id: values.unitId || undefined,
    category: values.category,
    vendor_id: values.vendorId || undefined,
    vendor_name: values.vendorId ? undefined : values.vendorName || undefined,
    amount_cents: dollarsToCents(values.amount),
    incurred_on: values.incurredOn,
    due_on: values.dueOn || undefined,
    description: values.description || undefined,
    tax_deductible: values.taxDeductible,
    is_recurring: values.isRecurring,
    recurrence_frequency: values.isRecurring ? values.recurrenceFrequency || undefined : undefined,
    attachment_id: values.attachmentId || undefined,
  };
  if (mode === 'create') {
    body.paid_on = values.paidOn || undefined;
    body.payment_method = values.paidOn ? values.paymentMethod || undefined : undefined;
  }
  return body;
}

export function toPaymentRequest(values: PaymentFormValues) {
  return {
    amount_cents: dollarsToCents(values.amount),
    paid_on: values.paidOn,
    method: values.method || undefined,
    reference: values.reference || undefined,
  };
}

export function toChargeRequest(values: ChargeFormValues) {
  return {
    unit_id: values.unitId,
    charge_type: values.chargeType,
    amount_cents: dollarsToCents(values.amount),
    description: values.description,
    due_on: values.dueOn,
  };
}

// ---- List params ----

export type RentRollSortKey = 'property' | 'unit' | 'dueOn' | 'outstanding';
const RENT_ROLL_SORT_TO_WIRE: Record<RentRollSortKey, string> = {
  property: 'property',
  unit: 'unit',
  dueOn: 'due_on',
  outstanding: 'outstanding',
};

export interface RentRollParams {
  propertyId?: string;
  leaseId?: string;
  status?: PaymentStatus | '';
  /** "YYYY-MM" */
  from?: string;
  to?: string;
  sort?: RentRollSortKey;
  order?: 'asc' | 'desc';
  limit: number;
  offset: number;
}

export type ExpenseSortKey = 'incurredOn' | 'amount' | 'property' | 'category';
const EXPENSE_SORT_TO_WIRE: Record<ExpenseSortKey, string> = {
  incurredOn: 'incurred_on',
  amount: 'amount',
  property: 'property',
  category: 'category',
};

export interface ExpenseListParams {
  search?: string;
  propertyId?: string;
  category?: ExpenseCategory | '';
  vendorId?: string;
  status?: ExpenseStatus | '';
  from?: string;
  to?: string;
  sort?: ExpenseSortKey;
  order?: 'asc' | 'desc';
  limit: number;
  offset: number;
}

export interface ChargeListParams {
  propertyId?: string;
  status?: PaymentStatus | '';
  limit: number;
  offset: number;
}

function query(params: Record<string, string | number | undefined>): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') q.set(k, String(v));
  }
  const s = q.toString();
  return s ? `?${s}` : '';
}

export const accountingApi = {
  async dashboard(period: DashboardPeriod): Promise<AccountingDashboard> {
    return toDashboard(await apiClient.get<DashboardWire>(`/api/v1/accounting/dashboard${query({ period })}`));
  },

  async getSettings(): Promise<AccountingSettings> {
    return toSettings(await apiClient.get<SettingsWire>('/api/v1/accounting/settings'));
  },

  async updateSettings(s: AccountingSettings): Promise<AccountingSettings> {
    return toSettings(
      await apiClient.put<SettingsWire>('/api/v1/accounting/settings', {
        late_fee_kind: s.lateFeeKind,
        late_fee_value: s.lateFeeValue,
        grace_days: s.graceDays,
        default_rent_due_day: s.defaultRentDueDay,
      }),
    );
  },

  async listRentRoll(params: RentRollParams): Promise<{ rows: RentRollRow[]; total: number }> {
    const { data, meta } = await apiClient.getWithMeta<RentRollWire[], ListMeta>(
      `/api/v1/accounting/rent-roll${query({
        property_id: params.propertyId,
        lease_id: params.leaseId,
        status: params.status,
        from: params.from,
        to: params.to,
        sort: params.sort ? RENT_ROLL_SORT_TO_WIRE[params.sort] : undefined,
        order: params.order,
        limit: params.limit,
        offset: params.offset,
      })}`,
    );
    return { rows: data.map(toRentRollRow), total: meta.total };
  },

  generateRent(period: string): Promise<{ rent_created: number; late_fee_created: number }> {
    return apiClient.post('/api/v1/accounting/rent-roll/generate', { period });
  },

  async recordLeasePayment(leaseId: string, values: PaymentFormValues): Promise<Payment[]> {
    const data = await apiClient.post<PaymentWire[]>(
      `/api/v1/accounting/rent-roll/leases/${leaseId}/payments`,
      toPaymentRequest(values),
    );
    return data.map(toPayment);
  },

  async listExpenses(params: ExpenseListParams): Promise<{ rows: LedgerRow[]; total: number }> {
    const { data, meta } = await apiClient.getWithMeta<LedgerWire[], ListMeta>(
      `/api/v1/accounting/expenses${query({
        search: params.search,
        property_id: params.propertyId,
        category: params.category,
        vendor_id: params.vendorId,
        status: params.status,
        from: params.from,
        to: params.to,
        sort: params.sort ? EXPENSE_SORT_TO_WIRE[params.sort] : undefined,
        order: params.order,
        limit: params.limit,
        offset: params.offset,
      })}`,
    );
    return { rows: data.map(toLedgerRow), total: meta.total };
  },

  async getExpense(id: string): Promise<LedgerRow> {
    return toLedgerRow(await apiClient.get<LedgerWire>(`/api/v1/accounting/expenses/${id}`));
  },

  async createExpense(values: ExpenseFormValues): Promise<LedgerRow> {
    return toLedgerRow(await apiClient.post<LedgerWire>('/api/v1/accounting/expenses', toExpenseRequest(values, 'create')));
  },

  async updateExpense(id: string, values: ExpenseFormValues): Promise<LedgerRow> {
    return toLedgerRow(
      await apiClient.put<LedgerWire>(`/api/v1/accounting/expenses/${id}`, toExpenseRequest(values, 'update')),
    );
  },

  async listCharges(params: ChargeListParams): Promise<{ rows: LedgerRow[]; total: number }> {
    const { data, meta } = await apiClient.getWithMeta<LedgerWire[], ListMeta>(
      `/api/v1/accounting/charges${query({
        property_id: params.propertyId,
        status: params.status,
        limit: params.limit,
        offset: params.offset,
      })}`,
    );
    return { rows: data.map(toLedgerRow), total: meta.total };
  },

  async createCharge(values: ChargeFormValues): Promise<LedgerRow> {
    return toLedgerRow(await apiClient.post<LedgerWire>('/api/v1/accounting/charges', toChargeRequest(values)));
  },

  async recordPayment(transactionId: string, values: PaymentFormValues): Promise<LedgerRow> {
    return toLedgerRow(
      await apiClient.post<LedgerWire>(`/api/v1/accounting/transactions/${transactionId}/payments`, toPaymentRequest(values)),
    );
  },

  async listPayments(transactionId: string): Promise<Payment[]> {
    const data = await apiClient.get<PaymentWire[]>(`/api/v1/accounting/transactions/${transactionId}/payments`);
    return data.map(toPayment);
  },

  async voidTransaction(id: string): Promise<void> {
    await apiClient.post(`/api/v1/accounting/transactions/${id}/void`);
  },

  async voidPayment(id: string): Promise<void> {
    await apiClient.post(`/api/v1/accounting/payments/${id}/void`);
  },

  async listAudit(transactionId: string): Promise<AuditEntry[]> {
    const data = await apiClient.get<
      { id: string; action: string; actor_id: string; changes: AuditEntry['changes']; created_at: string }[]
    >(`/api/v1/accounting/transactions/${transactionId}/audit`);
    return data.map((e) => ({ id: e.id, action: e.action, actorId: e.actor_id, changes: e.changes, createdAt: e.created_at }));
  },
};
