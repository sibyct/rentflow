// Display data for the Properties list + Add Property drawer. The
// backend's Property model only has address/unitCount/status/ownerId
// (see ../types) — no name, type, occupancy, rent collected, amenities,
// or documents. Rather than block the richer list/drawer design on a
// backend migration, this feature works against local mock rows, the
// same pattern DashboardPage and TopBar already use for data the API
// doesn't provide yet.

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
  status: PropertyRowStatus;
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

export const MOCK_PROPERTIES: PropertyRow[] = [
  {
    id: 'p1',
    name: 'Willow Creek Apartments',
    address: '214 Willow Creek Rd, Austin, TX 78745',
    type: 'Residential – Multi Unit',
    units: 24,
    occupancyPct: 92,
    collectedThisMonth: 28400,
    status: 'Active',
  },
  {
    id: 'p2',
    name: 'The Hendricks Building',
    address: '88 Hendricks Ave, Austin, TX 78702',
    type: 'Commercial',
    units: 6,
    occupancyPct: 100,
    collectedThisMonth: 19800,
    status: 'Active',
  },
  {
    id: 'p3',
    name: 'Maple & 9th Duplex',
    address: '902 Maple St, Austin, TX 78703',
    type: 'Residential – Multi Unit',
    units: 2,
    occupancyPct: 50,
    collectedThisMonth: 2150,
    status: 'Active',
  },
  {
    id: 'p4',
    name: 'Sable Point Townhomes',
    address: '47 Sable Point Ln, Round Rock, TX 78664',
    type: 'Mixed Use',
    units: 12,
    occupancyPct: 83,
    collectedThisMonth: 16020,
    status: 'Active',
  },
  {
    id: 'p5',
    name: 'Riverside Storage Depot',
    address: '500 Riverside Dr, Austin, TX 78741',
    type: 'Commercial',
    units: 1,
    occupancyPct: 0,
    collectedThisMonth: 0,
    status: 'Archived',
  },
];
