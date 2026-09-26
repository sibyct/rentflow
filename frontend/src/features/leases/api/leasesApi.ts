import { apiClient } from '@/api/client';
import type {
  DepositStatus,
  LeaseAuditEntry,
  LeaseDetailWithUnitProperty,
  LeaseDisplayStatus,
  LeaseDocument,
  LeaseDocumentCategory,
  LeaseRentHistoryEntry,
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
  proposed_rent?: number | null;
  proposed_end_date?: string;
  offer_sent_date?: string;
  termination_reason?: string;
  termination_notice_date?: string;
  renewed_into_lease_id?: string;
  renewed_from_lease_id?: string;
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
    proposedRent: wire.proposed_rent ?? null,
    proposedEndDate: wire.proposed_end_date ?? '',
    offerSentDate: wire.offer_sent_date ?? '',
    terminationReason: (wire.termination_reason as TerminationReason) ?? '',
    terminationNoticeDate: wire.termination_notice_date ?? '',
    renewedIntoLeaseId: wire.renewed_into_lease_id ?? '',
    renewedFromLeaseId: wire.renewed_from_lease_id ?? '',
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
    proposedRent: detail.proposedRent != null ? String(detail.proposedRent) : '',
    proposedEndDate: detail.proposedEndDate,
    offerSentDate: detail.offerSentDate,
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

// The update endpoint additionally accepts renewal_status/proposed_*/
// termination_reason/termination_notice_date/signed/signed_date —
// fields that only make sense once a lease already exists (you don't
// terminate a lease you're still creating), so they're absent from
// toCreateLeaseRequest above.
//
// monthly_rent is only ever included while the lease is still a draft
// (originalStatus, the status the lease had *before* this edit, not
// whatever the form's Status field currently holds) — once a lease has
// gone active, the backend rejects a direct monthly_rent edit outright
// (R3); a rent change on an active lease only ever goes through
// changeRent below. See LeaseFormDialog's monthlyRent field, which is
// disabled for exactly this reason once active.
function toUpdateLeaseRequest(values: LeaseFormValues, originalStatus: LeaseStatus) {
  const { monthly_rent, ...rest } = toCreateLeaseRequest(values);
  return {
    ...rest,
    monthly_rent: originalStatus === 'draft' ? monthly_rent : undefined,
    renewal_status: values.renewalStatus,
    proposed_rent: parseNumber(values.proposedRent),
    proposed_end_date: values.proposedEndDate || undefined,
    offer_sent_date: values.offerSentDate || undefined,
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

// Wire shapes for the R1-R20 lease-lifecycle endpoints — each stays
// private to this file, same rationale as LeaseWire above.
interface LeaseRentHistoryWire {
  id: string;
  lease_id: string;
  amount: number;
  effective_date: string;
  reason?: string;
  is_correction: boolean;
  created_by: string;
  created_at: string;
}

function toLeaseRentHistoryEntry(wire: LeaseRentHistoryWire): LeaseRentHistoryEntry {
  return {
    id: wire.id,
    leaseId: wire.lease_id,
    amount: wire.amount,
    effectiveDate: wire.effective_date,
    reason: wire.reason ?? '',
    isCorrection: wire.is_correction,
    createdBy: wire.created_by,
    createdAt: wire.created_at,
  };
}

interface LeaseAuditEntryWire {
  id: string;
  lease_id: string;
  action: string;
  actor_id: string;
  changes: Record<string, { old: unknown; new: unknown }>;
  created_at: string;
}

function toLeaseAuditEntry(wire: LeaseAuditEntryWire): LeaseAuditEntry {
  return {
    id: wire.id,
    leaseId: wire.lease_id,
    action: wire.action,
    actorId: wire.actor_id,
    changes: wire.changes,
    createdAt: wire.created_at,
  };
}

interface LeaseDocumentWire {
  id: string;
  lease_id: string;
  attachment_id: string;
  category: string;
  uploaded_by: string;
  uploaded_by_name: string;
  is_automated: boolean;
  filename: string;
  content_type: string;
  size_bytes: number;
  created_at: string;
}

function toLeaseDocument(wire: LeaseDocumentWire): LeaseDocument {
  return {
    id: wire.id,
    leaseId: wire.lease_id,
    attachmentId: wire.attachment_id,
    category: wire.category as LeaseDocumentCategory,
    uploadedBy: wire.uploaded_by,
    uploadedByName: wire.uploaded_by_name,
    isAutomated: wire.is_automated,
    filename: wire.filename,
    contentType: wire.content_type,
    sizeBytes: wire.size_bytes,
    createdAt: wire.created_at,
  };
}

export interface ChangeRentInput {
  amount: string;
  effectiveDate: string;
  reason: string;
  isCorrection: boolean;
}

export interface GenerateRenewalInput {
  startDate: string;
  endDate: string;
  monthlyRent: string;
  securityDeposit: string;
  rentDueDay: string;
}

export interface TerminateLeaseInput {
  terminationReason: Exclude<TerminationReason, ''>;
  terminationNoticeDate: string;
  moveOutDate: string;
  moveOutInspectionAttachmentId: string;
}

/** Backs the "Correct termination details" action — fixing a data-entry mistake in an already-terminated lease's termination record. Never a silent inline edit: `reason` is mandatory and every correction is audit-logged. */
export interface CorrectTerminationInput {
  terminationReason: Exclude<TerminationReason, ''>;
  terminationNoticeDate: string;
  moveOutDate: string;
  reason: string;
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

  update: (id: string, values: LeaseFormValues, originalStatus: LeaseStatus): Promise<LeaseDetailWithUnitProperty> =>
    apiClient
      .put<{ id: string }>(`/api/v1/leases/${id}`, toUpdateLeaseRequest(values, originalStatus))
      .then((r) => leasesApi.get(r.id)),

  delete: (id: string): Promise<void> => apiClient.delete<void>(`/api/v1/leases/${id}`),

  rentHistory: (id: string): Promise<LeaseRentHistoryEntry[]> =>
    apiClient.get<LeaseRentHistoryWire[]>(`/api/v1/leases/${id}/rent-history`).then((rows) => rows.map(toLeaseRentHistoryEntry)),

  /** The only way to change an active lease's effective rent — always a new dated amendment, never an overwrite (R3/R4). */
  changeRent: (id: string, input: ChangeRentInput): Promise<LeaseDetailWithUnitProperty> =>
    apiClient
      .post<{ id: string }>(`/api/v1/leases/${id}/rent-changes`, {
        amount: input.amount,
        effective_date: input.effectiveDate,
        reason: input.reason || undefined,
        is_correction: input.isCorrection,
      })
      .then((r) => leasesApi.get(r.id)),

  /** Turns an Accepted renewal into a new, real Lease record (R7) — the source lease's own record is retired, never rewritten. */
  generateRenewal: (id: string, input: GenerateRenewalInput): Promise<LeaseDetailWithUnitProperty> =>
    apiClient
      .post<{ id: string }>(`/api/v1/leases/${id}/renewal-generate`, {
        start_date: input.startDate,
        end_date: input.endDate || undefined,
        monthly_rent: parseNumber(input.monthlyRent),
        security_deposit: parseNumber(input.securityDeposit),
        rent_due_day: parseNumber(input.rentDueDay),
      })
      .then((r) => leasesApi.get(r.id)),

  /** The dedicated action for ending a lease — always sets the unit Vacant (R10), never a generic status edit. */
  terminate: (id: string, input: TerminateLeaseInput): Promise<LeaseDetailWithUnitProperty> =>
    apiClient
      .post<{ id: string }>(`/api/v1/leases/${id}/terminate`, {
        termination_reason: input.terminationReason,
        termination_notice_date: input.terminationNoticeDate,
        move_out_date: input.moveOutDate || undefined,
        move_out_inspection_attachment_id: input.moveOutInspectionAttachmentId || undefined,
      })
      .then((r) => leasesApi.get(r.id)),

  /** Fixes a mistake in an already-terminated lease's termination details — a separate action from terminate, requiring a reason and captured in the audit log; never a silent inline edit. */
  correctTermination: (id: string, input: CorrectTerminationInput): Promise<LeaseDetailWithUnitProperty> =>
    apiClient
      .post<{ id: string }>(`/api/v1/leases/${id}/termination-correction`, {
        termination_reason: input.terminationReason,
        termination_notice_date: input.terminationNoticeDate,
        move_out_date: input.moveOutDate || undefined,
        reason: input.reason,
      })
      .then((r) => leasesApi.get(r.id)),

  audit: (id: string): Promise<LeaseAuditEntry[]> =>
    apiClient.get<LeaseAuditEntryWire[]>(`/api/v1/leases/${id}/audit`).then((rows) => rows.map(toLeaseAuditEntry)),

  documents: (id: string): Promise<LeaseDocument[]> =>
    apiClient.get<LeaseDocumentWire[]>(`/api/v1/leases/${id}/documents`).then((rows) => rows.map(toLeaseDocument)),

  addDocument: (id: string, attachmentId: string, category: LeaseDocumentCategory): Promise<LeaseDocument> =>
    apiClient
      .post<LeaseDocumentWire>(`/api/v1/leases/${id}/documents`, { attachment_id: attachmentId, category })
      .then(toLeaseDocument),

  deleteDocument: (id: string, documentId: string): Promise<void> =>
    apiClient.delete<void>(`/api/v1/leases/${id}/documents/${documentId}`),
};
