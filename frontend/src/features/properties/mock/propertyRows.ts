// Display types for the Properties list, detail, and Add/Edit form —
// see api/propertiesApi.ts for the wire<->display mapping against the
// real backend. This module is named "mock" from an earlier phase where
// the feature worked against local rows instead of the API; the name
// stuck even though nothing here is mock data anymore.

export type PropertyType = 'Residential – Single Unit' | 'Residential – Multi Unit' | 'Commercial' | 'Mixed Use';
export type PropertyRowStatus = 'Active' | 'Onboarding' | 'Archived';

export interface PropertyRow {
  id: string;
  name: string;
  address: string;
  type: PropertyType;
  units: number;
  occupancyPct: number;
  collectedThisMonth: number;
  /** Actual unit rows behind this property (see features/units) — distinct from `units`, the manually-entered target count from the Add/Edit form. */
  unitCount: number;
  status: PropertyRowStatus;
}

/** Full record for the property detail/edit views — everything PropertyRow has, plus every field the Add/Edit form collects. */
export interface PropertyDetail {
  id: string;
  name: string;
  type: PropertyType;
  address: string;
  addressLine1: string;
  addressLine2: string;
  city: string;
  stateProvince: string;
  postalCode: string;
  country: string;
  units: number;
  ownership: 'owned' | 'managed' | '';
  ownerName: string;
  yearBuilt: number | null;
  /** ISO 'YYYY-MM-DD', or '' when unset. */
  onboardDate: string;
  amenities: string[];
  notes: string;
  status: PropertyRowStatus;
  occupancyPct: number;
  collectedThisMonth: number;
  unitCount: number;
  createdAt: string;
  updatedAt: string;
}

export const PROPERTY_TYPES: PropertyType[] = [
  'Residential – Single Unit',
  'Residential – Multi Unit',
  'Commercial',
  'Mixed Use',
];

export const AMENITIES = ['Parking', 'Laundry', 'Pool', 'Elevator', 'Pet-friendly', 'Gym', 'Storage', 'EV charging'] as const;

export const US_STATES = [
  'Alabama', 'Alaska', 'Arizona', 'Arkansas', 'California', 'Colorado', 'Connecticut', 'Delaware', 'Florida', 'Georgia',
  'Hawaii', 'Idaho', 'Illinois', 'Indiana', 'Iowa', 'Kansas', 'Kentucky', 'Louisiana', 'Maine', 'Maryland',
  'Massachusetts', 'Michigan', 'Minnesota', 'Mississippi', 'Missouri', 'Montana', 'Nebraska', 'Nevada', 'New Hampshire', 'New Jersey',
  'New Mexico', 'New York', 'North Carolina', 'North Dakota', 'Ohio', 'Oklahoma', 'Oregon', 'Pennsylvania', 'Rhode Island', 'South Carolina',
  'South Dakota', 'Tennessee', 'Texas', 'Utah', 'Vermont', 'Virginia', 'Washington', 'West Virginia', 'Wisconsin', 'Wyoming',
];

export const COUNTRIES = ['United States', 'Canada', 'United Kingdom', 'Australia'];

/** Defaults to the org's country — stands in for a real org-settings lookup. */
export const ORG_DEFAULT_COUNTRY = 'United States';
