import { z } from 'zod';
import { UNIT_TYPES, UNIT_STATUSES } from '../types';

// Numeric fields are strings in the form (like AddPropertyFormValues'
// yearBuilt) and parsed to numbers only at the API boundary — see
// unitsApi.ts's toCreateUnitRequest — so an empty/mid-typed field never
// fights a `z.number()` coercion while the user is still typing.
export const unitSchema = z.object({
  unitName: z.string().trim().min(1, 'Unit name is required.'),
  floor: z.string().trim().optional(),
  type: z.enum(UNIT_TYPES, { error: 'Select a unit type.' }),
  bedrooms: z.string().trim().optional(),
  bathrooms: z.string().trim().optional(),
  sqft: z.string().trim().optional(),
  furnished: z.union([z.enum(['unfurnished', 'furnished', 'partial']), z.literal('')]).optional(),
  status: z.enum(UNIT_STATUSES),
  marketRent: z.string().trim().optional(),
  currentRent: z.string().trim().optional(),
  securityDeposit: z.string().trim().optional(),
  rentDueDay: z.string().trim().optional(),
  // Freeform label, not a real tenant link — see types.ts / UnitDetail.
  tenantName: z.string().trim().optional(),
  notes: z.string().trim().optional(),
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
    currentRent: '',
    securityDeposit: '',
    rentDueDay: '',
    tenantName: '',
    notes: '',
  };
}
