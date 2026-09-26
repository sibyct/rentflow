import { z } from 'zod';
import { UNIT_TYPES, UNIT_STATUSES } from '../types';

// Numeric fields are strings in the form (like AddPropertyFormValues'
// yearBuilt) and parsed to numbers only at the API boundary — see
// unitsApi.ts's toCreateUnitRequest — so an empty/mid-typed field never
// fights a `z.number()` coercion while the user is still typing.
export const unitSchema = z
  .object({
    unitName: z.string().trim().min(1, 'Unit name is required.'),
    floor: z.string().trim().optional(),
    type: z.enum(UNIT_TYPES, { error: 'Select a unit type.' }),
    bedrooms: z.string().trim().optional(),
    bathrooms: z.string().trim().optional(),
    sqft: z.string().trim().optional(),
    furnished: z.union([z.enum(['unfurnished', 'furnished', 'partial']), z.literal('')]).optional(),
    status: z.enum(UNIT_STATUSES),
    // Current Rent has no field here — it's system-derived from the
    // unit's active lease (or shows "Vacant" with none) and is never
    // directly editable, see R1 in the Unit & Lease business rules.
    marketRent: z.string().trim().optional(),
    securityDeposit: z.string().trim().optional(),
    rentDueDay: z.string().trim().optional(),
    // Freeform label, not a real tenant link — see types.ts / UnitDetail.
    tenantName: z.string().trim().optional(),
    notes: z.string().trim().optional(),
  })
  // R20: every monetary field is validated non-negative before save —
  // mirrors the backend's own checks in UnitService.validateUnit/
  // UpdateUnit. Without this, a negative value round-trips to the
  // server and back as an unhelpful top-level error instead of an
  // inline field one.
  .refine((data) => !data.bedrooms || Number(data.bedrooms) >= 0, { message: 'Bedrooms cannot be negative.', path: ['bedrooms'] })
  .refine((data) => !data.bathrooms || Number(data.bathrooms) >= 0, { message: 'Bathrooms cannot be negative.', path: ['bathrooms'] })
  .refine((data) => !data.sqft || Number(data.sqft) > 0, { message: 'Square footage must be positive.', path: ['sqft'] })
  .refine((data) => !data.marketRent || Number(data.marketRent) >= 0, { message: 'Market rent cannot be negative.', path: ['marketRent'] })
  .refine((data) => !data.securityDeposit || Number(data.securityDeposit) >= 0, { message: 'Security deposit cannot be negative.', path: ['securityDeposit'] })
  .refine((data) => !data.rentDueDay || (Number(data.rentDueDay) >= 1 && Number(data.rentDueDay) <= 31), {
    message: 'Rent due day must be between 1 and 31.',
    path: ['rentDueDay'],
  });

export type UnitFormValues = z.infer<typeof unitSchema>;

export function unitDefaultValues(): UnitFormValues {
  return {
    unitName: '',
    floor: '',
    type: undefined as unknown as UnitFormValues['type'],
    bedrooms: '',
    bathrooms: '',
    sqft: '',
    furnished: '',
    status: 'vacant',
    marketRent: '',
    securityDeposit: '',
    rentDueDay: '',
    tenantName: '',
    notes: '',
  };
}
