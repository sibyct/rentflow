// Display types for the Units feature — the rentable things inside a
// Property (an apartment, a suite, or — for a residential_single_unit
// property — the property itself, represented by one implicit unit the
// backend creates automatically; see PropertyService.CreateProperty).
// Mirrors mock/propertyRows.ts's role in the properties feature, minus
// the "mock" part: units are real data from day one, no local-only
// fields waiting on a backend migration.

export type UnitType = 'studio' | '1br' | '2br' | '3br_plus' | 'commercial_suite' | 'other';

export const UNIT_TYPES: UnitType[] = ['studio', '1br', '2br', '3br_plus', 'commercial_suite', 'other'];

export const UNIT_TYPE_LABELS: Record<UnitType, string> = {
  studio: 'Studio',
  '1br': '1 Bedroom',
  '2br': '2 Bedroom',
  '3br_plus': '3+ Bedroom',
  commercial_suite: 'Commercial suite',
  other: 'Other',
};

export type UnitFurnished = 'unfurnished' | 'furnished' | 'partial' | '';

export const UNIT_FURNISHED_OPTIONS: Exclude<UnitFurnished, ''>[] = ['unfurnished', 'furnished', 'partial'];

export const UNIT_FURNISHED_LABELS: Record<Exclude<UnitFurnished, ''>, string> = {
  unfurnished: 'Unfurnished',
  furnished: 'Furnished',
  partial: 'Partially furnished',
};

export type UnitStatus = 'vacant' | 'occupied' | 'maintenance' | 'off_market';

export const UNIT_STATUSES: UnitStatus[] = ['vacant', 'occupied', 'maintenance', 'off_market'];

export const UNIT_STATUS_LABELS: Record<UnitStatus, string> = {
  vacant: 'Vacant',
  occupied: 'Occupied',
  maintenance: 'Maintenance',
  off_market: 'Off market',
};

export interface UnitRow {
  id: string;
  propertyId: string;
  unitName: string;
  floor: string;
  type: UnitType;
  bedrooms: number | null;
  bathrooms: number | null;
  status: UnitStatus;
  marketRent: number | null;
  currentRent: number | null;
  tenantName: string;
  /** When this unit most recently became vacant — empty when it isn't currently vacant, or became vacant before this field existed. */
  vacatedAt: string;
}

/** Full record for the unit detail/edit views — everything UnitRow has, plus every field the Add/Edit Unit form collects. */
export interface UnitDetail extends UnitRow {
  sqft: number | null;
  furnished: UnitFurnished;
  securityDeposit: number | null;
  rentDueDay: number | null;
  notes: string;
  createdAt: string;
  updatedAt: string;
}

/** A UnitRow decorated with its parent property's name — the shape the portfolio-wide global Units page needs for its clickable Property column (see GlobalUnitsTable). The property-scoped UnitsTable has no need for this since the property is already known from the page it's on. */
export interface UnitRowWithProperty extends UnitRow {
  propertyName: string;
}
