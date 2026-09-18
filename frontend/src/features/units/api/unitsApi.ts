import { apiClient } from '@/api/client';
import type { UnitDetail, UnitFurnished, UnitRow, UnitRowWithProperty, UnitStatus, UnitType } from '../types';
import type { UnitFormValues } from '../schemas/unitSchema';

// Wire shape exactly as internal/transport/http/dto/unit_dto.go's
// UnitResponse serializes it — see propertiesApi.ts's PropertyWire for
// why this stays private to this file.
interface UnitWire {
  id: string;
  property_id: string;
  unit_name: string;
  floor?: string;
  unit_type: string;
  bedrooms?: number | null;
  bathrooms?: number | null;
  sqft?: number | null;
  furnished?: string;
  status: string;
  market_rent?: number | null;
  current_rent?: number | null;
  security_deposit?: number | null;
  rent_due_day?: number | null;
  tenant_name?: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

function toUnitDetail(wire: UnitWire): UnitDetail {
  return {
    id: wire.id,
    propertyId: wire.property_id,
    unitName: wire.unit_name,
    floor: wire.floor ?? '',
    type: wire.unit_type as UnitType,
    bedrooms: wire.bedrooms ?? null,
    bathrooms: wire.bathrooms ?? null,
    sqft: wire.sqft ?? null,
    furnished: (wire.furnished as UnitFurnished) ?? '',
    status: wire.status as UnitStatus,
    marketRent: wire.market_rent ?? null,
    currentRent: wire.current_rent ?? null,
    securityDeposit: wire.security_deposit ?? null,
    rentDueDay: wire.rent_due_day ?? null,
    tenantName: wire.tenant_name ?? '',
    notes: wire.notes ?? '',
    createdAt: wire.created_at,
    updatedAt: wire.updated_at,
  };
}

// UnitRow is a strict subset of UnitDetail's fields, so every list row
// is just its detail projection — one converter, not two shapes to keep
// in sync.
function toUnitRow(wire: UnitWire): UnitRow {
  return toUnitDetail(wire);
}

interface UnitWithPropertyWire extends UnitWire {
  property_name: string;
}

function toUnitRowWithProperty(wire: UnitWithPropertyWire): UnitRowWithProperty {
  return { ...toUnitRow(wire), propertyName: wire.property_name };
}

interface UnitListMetaWire {
  total: number;
  limit: number;
  offset: number;
}

/** Prefills the Add/Edit Unit form from an existing unit — the reverse of toCreateUnitRequest below. */
export function toFormValues(detail: UnitDetail): UnitFormValues {
  return {
    unitName: detail.unitName,
    floor: detail.floor,
    type: detail.type,
    bedrooms: detail.bedrooms != null ? String(detail.bedrooms) : '',
    bathrooms: detail.bathrooms != null ? String(detail.bathrooms) : '',
    sqft: detail.sqft != null ? String(detail.sqft) : '',
    furnished: detail.furnished,
    status: detail.status,
    marketRent: detail.marketRent != null ? String(detail.marketRent) : '',
    currentRent: detail.currentRent != null ? String(detail.currentRent) : '',
    securityDeposit: detail.securityDeposit != null ? String(detail.securityDeposit) : '',
    rentDueDay: detail.rentDueDay != null ? String(detail.rentDueDay) : '',
    tenantName: detail.tenantName,
    notes: detail.notes,
  };
}

function parseNumber(value: string | undefined): number | undefined {
  if (!value || value.trim() === '') return undefined;
  const n = Number(value);
  return Number.isFinite(n) ? n : undefined;
}

function toCreateUnitRequest(values: UnitFormValues) {
  return {
    unit_name: values.unitName.trim(),
    floor: values.floor?.trim() || undefined,
    unit_type: values.type,
    bedrooms: parseNumber(values.bedrooms),
    bathrooms: parseNumber(values.bathrooms),
    sqft: parseNumber(values.sqft),
    furnished: values.furnished || undefined,
    status: values.status,
    market_rent: parseNumber(values.marketRent),
    current_rent: parseNumber(values.currentRent),
    security_deposit: parseNumber(values.securityDeposit),
    rent_due_day: parseNumber(values.rentDueDay),
    tenant_name: values.tenantName?.trim() || undefined,
    notes: values.notes?.trim() || undefined,
  };
}

/** One spreadsheet row in the bulk-add flow — a deliberately smaller field set than the full form (see BulkAddUnitsDialog). */
export interface BulkUnitRow {
  unitName: string;
  type: UnitType;
  bedrooms: string;
  bathrooms: string;
  marketRent: string;
}

function toBulkCreateUnitRequest(row: BulkUnitRow) {
  return {
    unit_name: row.unitName.trim(),
    unit_type: row.type,
    bedrooms: parseNumber(row.bedrooms),
    bathrooms: parseNumber(row.bathrooms),
    market_rent: parseNumber(row.marketRent),
  };
}

export type UnitPortfolioSortKey = 'rent' | 'status' | 'propertyName';
const SORT_TO_WIRE: Record<UnitPortfolioSortKey, string> = {
  rent: 'rent',
  status: 'status',
  propertyName: 'property_name',
};

export interface UnitPortfolioListParams {
  search?: string;
  propertyId?: string;
  status?: UnitStatus | '';
  type?: UnitType | '';
  sort?: UnitPortfolioSortKey;
  order?: 'asc' | 'desc';
  limit: number;
  offset: number;
}

export interface UnitPortfolioListResult {
  units: UnitRowWithProperty[];
  total: number;
}

function buildPortfolioListQuery(params: UnitPortfolioListParams): string {
  const q = new URLSearchParams();
  if (params.search) q.set('search', params.search);
  if (params.propertyId) q.set('property_id', params.propertyId);
  if (params.status) q.set('status', params.status);
  if (params.type) q.set('type', params.type);
  if (params.sort) q.set('sort', SORT_TO_WIRE[params.sort]);
  if (params.order) q.set('order', params.order);
  q.set('limit', String(params.limit));
  q.set('offset', String(params.offset));
  return q.toString();
}

export const unitsApi = {
  list: (propertyId: string): Promise<UnitRow[]> =>
    apiClient.get<UnitWire[]>(`/api/v1/properties/${propertyId}/units`).then((rows) => rows.map(toUnitRow)),

  /** The portfolio-wide list behind the global /units page — every unit across every property, unlike `list` above which is scoped to one property. */
  listForOwner: (params: UnitPortfolioListParams): Promise<UnitPortfolioListResult> =>
    apiClient
      .getWithMeta<UnitWithPropertyWire[], UnitListMetaWire>(`/api/v1/units?${buildPortfolioListQuery(params)}`)
      .then(({ data, meta }) => ({ units: data.map(toUnitRowWithProperty), total: meta.total })),

  get: (unitId: string): Promise<UnitDetail> =>
    apiClient.get<UnitWire>(`/api/v1/units/${unitId}`).then(toUnitDetail),

  create: (propertyId: string, values: UnitFormValues): Promise<UnitDetail> =>
    apiClient.post<UnitWire>(`/api/v1/properties/${propertyId}/units`, toCreateUnitRequest(values)).then(toUnitDetail),

  createBulk: (propertyId: string, rows: BulkUnitRow[]): Promise<UnitDetail[]> =>
    apiClient
      .post<UnitWire[]>(`/api/v1/properties/${propertyId}/units/bulk`, { units: rows.map(toBulkCreateUnitRequest) })
      .then((list) => list.map(toUnitDetail)),

  // Sends the whole form as a full replace, same rationale as
  // propertiesApi.ts's update: the edit drawer always shows every field.
  update: (unitId: string, values: UnitFormValues): Promise<UnitDetail> =>
    apiClient.put<UnitWire>(`/api/v1/units/${unitId}`, toCreateUnitRequest(values)).then(toUnitDetail),

  delete: (unitId: string): Promise<void> => apiClient.delete<void>(`/api/v1/units/${unitId}`),
};
