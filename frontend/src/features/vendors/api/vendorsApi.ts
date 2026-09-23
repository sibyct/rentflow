import { apiClient } from '@/api/client';
import type { WorkOrderCategory } from '@/features/maintenance/types';
import type { InsuranceStatus, VendorDetail, VendorRateType, VendorPaymentTerms, VendorRow, VendorSortKey, VendorSpendSummary } from '../types';
import type { VendorFormValues } from '../schemas/vendorSchema';

export type { VendorSortKey } from '../types';

// Wire shape exactly as internal/transport/http/dto/vendor_dto.go's
// VendorResponse/VendorWithStatsResponse serialize it.
interface VendorWire {
  id: string;
  company_name: string;
  categories: string[];
  contact_person?: string;
  phone?: string;
  email?: string;
  address?: string;
  serves_all_properties: boolean;
  insurance_expiry?: string;
  insurance_status: string;
  license_number?: string;
  license_expiry?: string;
  coi_attachment_id?: string;
  tax_doc_attachment_id?: string;
  rate_type?: string;
  rate_amount?: number | null;
  payment_terms?: string;
  internal_notes?: string;
  active: boolean;
  created_at: string;
  updated_at: string;
  open_work_orders?: number;
  average_rating?: number | null;
  properties_served_count?: number;
}

interface VendorListMetaWire {
  total: number;
  limit: number;
  offset: number;
}

interface VendorSpendSummaryWire {
  this_month: number;
  year_to_date: number;
}

function toVendorRow(wire: VendorWire): VendorRow {
  return {
    id: wire.id,
    companyName: wire.company_name,
    categories: wire.categories as WorkOrderCategory[],
    contactPerson: wire.contact_person ?? '',
    phone: wire.phone ?? '',
    email: wire.email ?? '',
    address: wire.address ?? '',
    servesAllProperties: wire.serves_all_properties,
    insuranceExpiry: wire.insurance_expiry ?? '',
    insuranceStatus: wire.insurance_status as InsuranceStatus,
    licenseNumber: wire.license_number ?? '',
    licenseExpiry: wire.license_expiry ?? '',
    coiAttachmentId: wire.coi_attachment_id ?? '',
    taxDocAttachmentId: wire.tax_doc_attachment_id ?? '',
    rateType: (wire.rate_type as VendorRateType) ?? '',
    rateAmount: wire.rate_amount ?? null,
    paymentTerms: (wire.payment_terms as VendorPaymentTerms) ?? '',
    internalNotes: wire.internal_notes ?? '',
    active: wire.active,
    createdAt: wire.created_at,
    updatedAt: wire.updated_at,
    openWorkOrders: wire.open_work_orders ?? 0,
    averageRating: wire.average_rating ?? null,
    propertiesServedCount: wire.properties_served_count ?? 0,
  };
}

/** Prefills the vendor form from an existing record — the reverse of toVendorRequest below. propertiesServed comes from a separate fetch (see vendorsApi.getPropertiesServed), not the vendor record itself. */
export function toFormValues(detail: VendorDetail, propertiesServed: string[]): VendorFormValues {
  return {
    companyName: detail.companyName,
    categories: detail.categories,
    contactPerson: detail.contactPerson,
    phone: detail.phone,
    email: detail.email,
    address: detail.address,
    servesAllProperties: detail.servesAllProperties,
    propertiesServed,
    insuranceExpiry: detail.insuranceExpiry,
    licenseNumber: detail.licenseNumber,
    licenseExpiry: detail.licenseExpiry,
    coiAttachmentId: detail.coiAttachmentId,
    taxDocAttachmentId: detail.taxDocAttachmentId,
    rateType: detail.rateType,
    rateAmount: detail.rateAmount != null ? String(detail.rateAmount) : '',
    paymentTerms: detail.paymentTerms,
    internalNotes: detail.internalNotes,
    active: detail.active,
  };
}

function parseNumber(value: string | undefined): number | undefined {
  if (!value || value.trim() === '') return undefined;
  const n = Number(value);
  return Number.isFinite(n) ? n : undefined;
}

function toVendorRequest(values: VendorFormValues) {
  return {
    company_name: values.companyName.trim(),
    categories: values.categories,
    contact_person: values.contactPerson?.trim() || undefined,
    phone: values.phone?.trim() || undefined,
    email: values.email?.trim() || undefined,
    address: values.address?.trim() || undefined,
    serves_all_properties: values.servesAllProperties,
    properties_served: values.servesAllProperties ? [] : values.propertiesServed,
    insurance_expiry: values.insuranceExpiry || undefined,
    license_number: values.licenseNumber?.trim() || undefined,
    license_expiry: values.licenseExpiry || undefined,
    // Always sent, even when empty — see workOrdersApi.ts's
    // toWorkOrderRequest for why (same empty-means-"clear" update
    // contract as UpdateVendorRequest.COIAttachmentID).
    coi_attachment_id: values.coiAttachmentId?.trim() ?? '',
    tax_doc_attachment_id: values.taxDocAttachmentId?.trim() ?? '',
    rate_type: values.rateType || undefined,
    rate_amount: parseNumber(values.rateAmount),
    payment_terms: values.paymentTerms || undefined,
    internal_notes: values.internalNotes?.trim() || undefined,
  };
}

function toUpdateVendorRequest(values: VendorFormValues) {
  return { ...toVendorRequest(values), active: values.active };
}

const SORT_TO_WIRE: Record<VendorSortKey, string> = {
  name: 'name',
  rating: 'rating',
};

export interface VendorListParams {
  search?: string;
  category?: WorkOrderCategory | '';
  active?: boolean;
  insuranceStatus?: InsuranceStatus | '';
  sort?: VendorSortKey;
  order?: 'asc' | 'desc';
  limit: number;
  offset: number;
}

export interface VendorListResult {
  vendors: VendorRow[];
  total: number;
}

function buildListQuery(params: VendorListParams): string {
  const q = new URLSearchParams();
  if (params.search) q.set('search', params.search);
  if (params.category) q.set('category', params.category);
  if (params.active != null) q.set('active', String(params.active));
  if (params.insuranceStatus) q.set('insurance_status', params.insuranceStatus);
  if (params.sort) q.set('sort', SORT_TO_WIRE[params.sort]);
  if (params.order) q.set('order', params.order);
  q.set('limit', String(params.limit));
  q.set('offset', String(params.offset));
  return q.toString();
}

export const vendorsApi = {
  list: (params: VendorListParams): Promise<VendorListResult> =>
    apiClient
      .getWithMeta<VendorWire[], VendorListMetaWire>(`/api/v1/vendors?${buildListQuery(params)}`)
      .then(({ data, meta }) => ({ vendors: data.map(toVendorRow), total: meta.total })),

  listForCategory: (category: WorkOrderCategory): Promise<VendorRow[]> =>
    apiClient.get<VendorWire[]>(`/api/v1/vendors/by-category?category=${encodeURIComponent(category)}`).then((rows) => rows.map(toVendorRow)),

  get: (id: string): Promise<VendorDetail> => apiClient.get<VendorWire>(`/api/v1/vendors/${id}`).then(toVendorRow),

  getPropertiesServed: (id: string): Promise<string[]> =>
    apiClient.get<{ property_ids: string[] }>(`/api/v1/vendors/${id}/properties`).then((r) => r.property_ids),

  getSpendSummary: (id: string): Promise<VendorSpendSummary> =>
    apiClient.get<VendorSpendSummaryWire>(`/api/v1/vendors/${id}/spend`).then((s) => ({ thisMonth: s.this_month, yearToDate: s.year_to_date })),

  // The create/update endpoints return a plain VendorResponse (no
  // stats), so re-fetch the WithStats record after writing — same
  // pattern as workOrdersApi/leasesApi's create/update.
  create: (values: VendorFormValues): Promise<VendorDetail> =>
    apiClient.post<VendorWire>('/api/v1/vendors', toVendorRequest(values)).then((wire) => vendorsApi.get(wire.id)),

  update: (id: string, values: VendorFormValues): Promise<VendorDetail> =>
    apiClient.put<VendorWire>(`/api/v1/vendors/${id}`, toUpdateVendorRequest(values)).then(() => vendorsApi.get(id)),

  delete: (id: string): Promise<void> => apiClient.delete<void>(`/api/v1/vendors/${id}`),
};
