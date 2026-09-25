import { apiClient } from '@/api/client';
import type {
  DepositStatus,
  LeaseDetailWithUnitProperty,
  LeaseDisplayStatus,
  LeaseRowWithUnitProperty,
  LeaseStatus,
  LeaseType,
  RenewalStatus,
  TerminationReason,
} from '../types';
import type { LeaseFormValues } from '../schemas/leaseSchema';

// Wire shape exactly as internal/transport/http/dto/lease_dto.go's
// LeaseWithUnitPropertyResponse serializes it — see propertiesApi.ts's
// PropertyWire for why this stays private to this file.
interface LeaseWire {
  id: string;
  unit_id: string;
  lease_type: string;
  status: string;
  display_status: string;
  start_date: string;
  end_date?: string;
  move_in_date?: string;
  move_out_date?: string;
  monthly_rent: number;
  security_deposit?: number | null;
  deposit_status?: string;
  rent_due_day?: number | null;
  late_fee_amount?: number | null;
  late_fee_grace_days?: number | null;
  primary_resident_name: string;
  primary_resident_phone?: string;
  primary_resident_email?: string;
  co_residents: string[];
  emergency_contact?: string;
  renewal_status: string;
  termination_reason?: string;
  termination_notice_date?: string;
  signed: boolean;
  signed_date?: string;
  notes?: string;
  created_at: string;
  updated_at: string;
  unit_name: string;
  property_id: string;
  property_name: string;
}

interface LeaseListMetaWire {
  total: number;
  limit: number;
  offset: number;
}

function toLeaseDetail(wire: LeaseWire): LeaseDetailWithUnitProperty {
  return {
    id: wire.id,
    unitId: wire.unit_id,
    leaseType: wire.lease_type as LeaseType,
    status: wire.status as LeaseStatus,
    displayStatus: wire.display_status as LeaseDisplayStatus,
    startDate: wire.start_date,
    endDate: wire.end_date ?? '',
    moveInDate: wire.move_in_date ?? '',
    moveOutDate: wire.move_out_date ?? '',
    monthlyRent: wire.monthly_rent,
    securityDeposit: wire.security_deposit ?? null,
    depositStatus: (wire.deposit_status as DepositStatus) ?? '',
    rentDueDay: wire.rent_due_day ?? null,
    lateFeeAmount: wire.late_fee_amount ?? null,
    lateFeeGraceDays: wire.late_fee_grace_days ?? null,
    primaryResidentName: wire.primary_resident_name,
    primaryResidentPhone: wire.primary_resident_phone ?? '',
    primaryResidentEmail: wire.primary_resident_email ?? '',
    coResidents: wire.co_residents,
    emergencyContact: wire.emergency_contact ?? '',
    renewalStatus: wire.renewal_status as RenewalStatus,
    terminationReason: (wire.termination_reason as TerminationReason) ?? '',
    terminationNoticeDate: wire.termination_notice_date ?? '',
    signed: wire.signed,
    signedDate: wire.signed_date ?? '',
    notes: wire.notes ?? '',
    createdAt: wire.created_at,
    updatedAt: wire.updated_at,
    unitName: wire.unit_name,
    propertyId: wire.property_id,
    propertyName: wire.property_name,
  };
}

function toLeaseRow(wire: LeaseWire): LeaseRowWithUnitProperty {
  return toLeaseDetail(wire);
}

/** Prefills the Add/Edit Lease form from an existing lease — the reverse of toCreateLeaseRequest below. */
export function toFormValues(detail: LeaseDetailWithUnitProperty): LeaseFormValues {
  return {
    leaseType: detail.leaseType,
    status: detail.status,
    startDate: detail.startDate,
    endDate: detail.endDate,
    moveInDate: detail.moveInDate,
    moveOutDate: detail.moveOutDate,
    monthlyRent: String(detail.monthlyRent),
    securityDeposit: detail.securityDeposit != null ? String(detail.securityDeposit) : '',
    depositStatus: detail.depositStatus,
    rentDueDay: detail.rentDueDay != null ? String(detail.rentDueDay) : '',
    lateFeeAmount: detail.lateFeeAmount != null ? String(detail.lateFeeAmount) : '',
    lateFeeGraceDays: detail.lateFeeGraceDays != null ? String(detail.lateFeeGraceDays) : '',
    primaryResidentName: detail.primaryResidentName,
    primaryResidentPhone: detail.primaryResidentPhone,
    primaryResidentEmail: detail.primaryResidentEmail,
    coResidents: detail.coResidents.join(', '),
    emergencyContact: detail.emergencyContact,
    renewalStatus: detail.renewalStatus,
    terminationReason: detail.terminationReason,
    terminationNoticeDate: detail.terminationNoticeDate,
    signed: detail.signed,
    signedDate: detail.signedDate,
    notes: detail.notes,
  };
}

function parseNumber(value: string | undefined): number | undefined {
  if (!value || value.trim() === '') return undefined;
  const n = Number(value);
  return Number.isFinite(n) ? n : undefined;
}

function toCreateLeaseRequest(values: LeaseFormValues) {
  return {
    lease_type: values.leaseType,
    status: values.status,
    start_date: values.startDate,
    end_date: values.endDate || undefined,
    move_in_date: values.moveInDate || undefined,
    move_out_date: values.moveOutDate || undefined,
    monthly_rent: parseNumber(values.monthlyRent) ?? 0,
    security_deposit: parseNumber(values.securityDeposit),
    deposit_status: values.depositStatus || undefined,
    rent_due_day: parseNumber(values.rentDueDay),
    late_fee_amount: parseNumber(values.lateFeeAmount),
    late_fee_grace_days: parseNumber(values.lateFeeGraceDays),
    primary_resident_name: values.primaryResidentName.trim(),
    primary_resident_phone: values.primaryResidentPhone?.trim() || undefined,
    primary_resident_email: values.primaryResidentEmail?.trim() || undefined,
    co_residents: values.coResidents
      ? values.coResidents
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean)
      : [],
    emergency_contact: values.emergencyContact?.trim() || undefined,
    notes: values.notes?.trim() || undefined,
  };
}

// The update endpoint additionally accepts renewal_status/
// termination_reason/termination_notice_date/signed/signed_date —
// fields that only make sense once a lease already exists (you don't
// terminate a lease you're still creating), so they're absent from
// toCreateLeaseRequest above.
function toUpdateLeaseRequest(values: LeaseFormValues) {
  return {
    ...toCreateLeaseRequest(values),
    renewal_status: values.renewalStatus,
    termination_reason: values.terminationReason || undefined,
    termination_notice_date: values.terminationNoticeDate || undefined,
    signed: values.signed,
    signed_date: values.signedDate || undefined,
  };
}

export type LeasePortfolioSortKey = 'startDate' | 'endDate' | 'monthlyRent' | 'status';
const SORT_TO_WIRE: Record<LeasePortfolioSortKey, string> = {
  startDate: 'start_date',
  endDate: 'end_date',
  monthlyRent: 'monthly_rent',
  status: 'status',
};

export interface LeasePortfolioListParams {
  search?: string;
  propertyId?: string;
  unitId?: string;
  status?: LeaseDisplayStatus | '';
  sort?: LeasePortfolioSortKey;
  order?: 'asc' | 'desc';
  limit: number;
  offset: number;
}

export interface LeasePortfolioListResult {
  leases: LeaseRowWithUnitProperty[];
  total: number;
}

function buildPortfolioListQuery(params: LeasePortfolioListParams): string {
  const q = new URLSearchParams();
  if (params.search) q.set('search', params.search);
  if (params.propertyId) q.set('property_id', params.propertyId);
  if (params.unitId) q.set('unit_id', params.unitId);
  if (params.status) q.set('status', params.status);
  if (params.sort) q.set('sort', SORT_TO_WIRE[params.sort]);
  if (params.order) q.set('order', params.order);
  q.set('limit', String(params.limit));
  q.set('offset', String(params.offset));
  return q.toString();
}

export const leasesApi = {
  listForOwner: (params: LeasePortfolioListParams): Promise<LeasePortfolioListResult> =>
    apiClient
      .getWithMeta<LeaseWire[], LeaseListMetaWire>(
        `/api/v1/leases?${buildPortfolioListQuery(params)}`,
      )
      .then(({ data, meta }) => ({ leases: data.map(toLeaseRow), total: meta.total })),

  get: (id: string): Promise<LeaseDetailWithUnitProperty> =>
    apiClient.get<LeaseWire>(`/api/v1/leases/${id}`).then(toLeaseDetail),

  /** unitId comes from context (the unit page/section the flow was opened from), never a form field — mirrors unitsApi.ts's create(propertyId, ...). */
  create: (unitId: string, values: LeaseFormValues): Promise<LeaseDetailWithUnitProperty> =>
    apiClient
      .post<{ id: string }>(`/api/v1/units/${unitId}/leases`, toCreateLeaseRequest(values))
      .then((r) => leasesApi.get(r.id)),

  update: (id: string, values: LeaseFormValues): Promise<LeaseDetailWithUnitProperty> =>
    apiClient
      .put<{ id: string }>(`/api/v1/leases/${id}`, toUpdateLeaseRequest(values))
      .then((r) => leasesApi.get(r.id)),

  delete: (id: string): Promise<void> => apiClient.delete<void>(`/api/v1/leases/${id}`),
};
