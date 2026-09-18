import { apiClient } from '@/api/client';
import type { PropertyDetail, PropertyRow, PropertyRowStatus, PropertyType } from '../mock/propertyRows';
import type { AddPropertyFormValues } from '../schemas/addPropertySchema';

const DEFAULT_PROPERTY_TYPE: PropertyType = 'Residential – Single Unit';

// Wire shapes exactly as internal/transport/http/dto/property_dto.go
// serializes them (snake_case, backend enum values). These stay private
// to this file; every other module in the app works with PropertyRow
// (camelCase, display-label enums) from ../mock/propertyRows.
interface PropertyWire {
  id: string;
  name: string;
  type: string;
  address: string;
  units: number;
  status: string;
  occupancy_pct: number;
  collected_this_month: number;
  unit_count: number;
}

interface PropertyListMetaWire {
  total: number;
  limit: number;
  offset: number;
}

// The full shape GET/PUT /api/v1/properties/:id return — everything
// PropertyWire has plus every field the Add/Edit form collects. Kept
// separate from PropertyWire (rather than making that one bigger)
// since the list endpoint never sends these fields.
interface PropertyDetailWire {
  id: string;
  name: string;
  type: string;
  address: string;
  address_line1: string;
  address_line2?: string;
  city?: string;
  state_province?: string;
  postal_code?: string;
  country?: string;
  units: number;
  ownership?: string;
  owner_name?: string;
  year_built?: number | null;
  onboard_date?: string;
  amenities: string[];
  notes?: string;
  status: string;
  occupancy_pct: number;
  collected_this_month: number;
  unit_count: number;
  created_at: string;
  updated_at: string;
}

// PROPERTY_TYPES / PropertyRowStatus are display labels the rest of the
// app (filters, the add-property form) already works with — these maps
// are the one place that translates them to/from the backend's
// domain.PropertyType / domain.PropertyStatus wire values.
const TYPE_TO_WIRE: Record<PropertyType, string> = {
  'Residential – Single Unit': 'residential_single_unit',
  'Residential – Multi Unit': 'residential_multi_unit',
  Commercial: 'commercial',
  'Mixed Use': 'mixed_use',
};
const WIRE_TO_TYPE: Record<string, PropertyType> = Object.fromEntries(
  Object.entries(TYPE_TO_WIRE).map(([label, wire]) => [wire, label as PropertyType]),
);

const STATUS_TO_WIRE: Record<PropertyRowStatus, string> = {
  Onboarding: 'onboarding',
  Active: 'active',
  Archived: 'archived',
};
const WIRE_TO_STATUS: Record<string, PropertyRowStatus> = Object.fromEntries(
  Object.entries(STATUS_TO_WIRE).map(([label, wire]) => [wire, label as PropertyRowStatus]),
);

export type PropertySortKey = 'name' | 'type' | 'units' | 'status' | 'createdAt';
const SORT_TO_WIRE: Record<PropertySortKey, string> = {
  name: 'name',
  type: 'type',
  units: 'units',
  status: 'status',
  createdAt: 'created_at',
};

function toPropertyRow(wire: PropertyWire): PropertyRow {
  return {
    id: wire.id,
    name: wire.name,
    address: wire.address,
    type: WIRE_TO_TYPE[wire.type] ?? DEFAULT_PROPERTY_TYPE,
    units: wire.units,
    occupancyPct: wire.occupancy_pct,
    collectedThisMonth: wire.collected_this_month,
    unitCount: wire.unit_count,
    status: WIRE_TO_STATUS[wire.status] ?? 'Onboarding',
  };
}

function toPropertyDetail(wire: PropertyDetailWire): PropertyDetail {
  return {
    id: wire.id,
    name: wire.name,
    type: WIRE_TO_TYPE[wire.type] ?? DEFAULT_PROPERTY_TYPE,
    address: wire.address,
    addressLine1: wire.address_line1,
    addressLine2: wire.address_line2 ?? '',
    city: wire.city ?? '',
    stateProvince: wire.state_province ?? '',
    postalCode: wire.postal_code ?? '',
    country: wire.country ?? '',
    units: wire.units,
    ownership: wire.ownership === 'owned' || wire.ownership === 'managed' ? wire.ownership : '',
    ownerName: wire.owner_name ?? '',
    yearBuilt: wire.year_built ?? null,
    onboardDate: wire.onboard_date ?? '',
    amenities: wire.amenities,
    notes: wire.notes ?? '',
    status: WIRE_TO_STATUS[wire.status] ?? 'Onboarding',
    occupancyPct: wire.occupancy_pct,
    collectedThisMonth: wire.collected_this_month,
    unitCount: wire.unit_count,
    createdAt: wire.created_at,
    updatedAt: wire.updated_at,
  };
}

/** Prefills the Add/Edit form from an existing property — the reverse of toCreatePropertyRequest below. */
export function toFormValues(detail: PropertyDetail): AddPropertyFormValues {
  return {
    name: detail.name,
    type: detail.type,
    fullAddress: detail.address,
    addressLine1: detail.addressLine1,
    addressLine2: detail.addressLine2,
    city: detail.city,
    stateProvince: detail.stateProvince,
    postalCode: detail.postalCode,
    country: detail.country,
    units: detail.units,
    ownership: detail.ownership,
    ownerName: detail.ownerName,
    yearBuilt: detail.yearBuilt ? String(detail.yearBuilt) : '',
    onboardDate: detail.onboardDate,
    amenities: detail.amenities,
    notes: detail.notes,
  };
}

// A 4-digit year the user is still mid-typing (or left blank) shouldn't
// be sent as year_built — the backend only accepts a complete,
// plausible year (1800-2100) or no value at all.
function parseYearBuilt(value: string | undefined): number | undefined {
  if (!value || !/^\d{4}$/.test(value)) return undefined;
  const year = Number(value);
  return year >= 1800 && year <= 2100 ? year : undefined;
}

function toCreatePropertyRequest(values: AddPropertyFormValues) {
  return {
    name: values.name.trim(),
    type: TYPE_TO_WIRE[values.type],
    // address_line1 is the backend's required field; fall back to the
    // freeform fullAddress the user typed/picked when the autocomplete
    // never populated the structured sub-fields (see
    // AddressAutocompleteField).
    address_line1: (values.addressLine1?.trim() || values.fullAddress.trim()),
    address_line2: values.addressLine2?.trim() || undefined,
    city: values.city?.trim() || undefined,
    state_province: values.stateProvince?.trim() || undefined,
    postal_code: values.postalCode?.trim() || undefined,
    country: values.country?.trim() || undefined,
    units: values.units,
    ownership: values.ownership || undefined,
    owner_name: values.ownerName?.trim() || undefined,
    year_built: parseYearBuilt(values.yearBuilt),
    onboard_date: values.onboardDate || undefined,
    amenities: values.amenities,
    notes: values.notes?.trim() || undefined,
  };
}

export interface PropertyListParams {
  search?: string;
  type?: PropertyType | '';
  status?: PropertyRowStatus | '';
  sort?: PropertySortKey;
  order?: 'asc' | 'desc';
  limit: number;
  offset: number;
}

export interface PropertyListResult {
  properties: PropertyRow[];
  total: number;
}

function buildListQuery(params: PropertyListParams): string {
  const q = new URLSearchParams();
  // search never matches `type` — backend/internal/repository/postgres/
  // property_repository.go's List() only ILIKEs name/address_line1/city
  // — so this and the Type dropdown filter on genuinely disjoint fields,
  // not two controls quietly doing the same job.
  if (params.search) q.set('search', params.search);
  if (params.type) q.set('type', TYPE_TO_WIRE[params.type]);
  if (params.status) q.set('status', STATUS_TO_WIRE[params.status]);
  if (params.sort) q.set('sort', SORT_TO_WIRE[params.sort]);
  if (params.order) q.set('order', params.order);
  q.set('limit', String(params.limit));
  q.set('offset', String(params.offset));
  return q.toString();
}

export const propertiesApi = {
  list: (params: PropertyListParams): Promise<PropertyListResult> =>
    apiClient
      .getWithMeta<PropertyWire[], PropertyListMetaWire>(`/api/v1/properties?${buildListQuery(params)}`)
      .then(({ data, meta }) => ({ properties: data.map(toPropertyRow), total: meta.total })),

  create: (values: AddPropertyFormValues): Promise<PropertyRow> =>
    apiClient.post<PropertyWire>('/api/v1/properties', toCreatePropertyRequest(values)).then(toPropertyRow),

  get: (id: string): Promise<PropertyDetail> =>
    apiClient.get<PropertyDetailWire>(`/api/v1/properties/${id}`).then(toPropertyDetail),

  // Sends the whole form as a full replace, not a partial patch — the
  // edit modal always shows every field, so there's nothing to diff.
  // The backend's UpdatePropertyRequest fields are all individually
  // optional (a *string/*int pointer each), but toCreatePropertyRequest's
  // output already matches its field names/shapes, so it works for both.
  update: (id: string, values: AddPropertyFormValues): Promise<PropertyDetail> =>
    apiClient.put<PropertyDetailWire>(`/api/v1/properties/${id}`, toCreatePropertyRequest(values)).then(toPropertyDetail),

  // Backs both the per-row kebab menu's Archive/Restore and the
  // multi-select bulk-actions toolbar's Archive — a single-id array is
  // just the n=1 case of the same bulk endpoint, so there's no separate
  // single-property status endpoint to call.
  updateStatus: (ids: string[], status: PropertyRowStatus): Promise<number> =>
    apiClient
      .patch<{ updated: number }>('/api/v1/properties/status', { ids, status: STATUS_TO_WIRE[status] })
      .then((r) => r.updated),
};
