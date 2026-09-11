import { z } from 'zod';
import { PROPERTY_TYPES, ORG_DEFAULT_COUNTRY } from '../mock/propertyRows';

// Client-side-only form: the fields here (type, city/state/postal,
// ownership, amenities, ...) aren't part of the backend's Property DTO
// yet — see mock/propertyRows.ts for why this drawer works against
// local state instead of CreatePropertyInput/createPropertySchema.
export const addPropertySchema = z.object({
  name: z.string().trim().min(1, 'Property name is required.'),
  type: z.enum(PROPERTY_TYPES, { error: 'Select a property type.' }),
  addressLine1: z.string().trim().min(1, 'Street address is required.'),
  addressLine2: z.string().trim().optional(),
  city: z.string().trim().min(1, 'City is required.'),
  stateProvince: z.string().trim().min(1, 'Select a state or province.'),
  postalCode: z.string().trim().min(2, 'Postal code is required.'),
  country: z.string().trim().min(1, 'Select a country.'),
  units: z
    .number({ error: 'Enter at least 1 unit.' })
    .int('Enter a whole number of units.')
    .min(1, 'Enter at least 1 unit.'),
  ownership: z.union([z.enum(['owned', 'managed']), z.literal('')]).optional(),
  ownerName: z.string().trim().optional(),
  yearBuilt: z.string().trim().optional(),
  onboardDate: z.string().trim().optional(),
  amenities: z.array(z.string()).default([]),
  notes: z.string().trim().optional(),
});

export type AddPropertyFormValues = z.infer<typeof addPropertySchema>;

export function addPropertyDefaultValues(): AddPropertyFormValues {
  const today = new Date();
  const iso = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`;
  return {
    name: '',
    type: undefined as unknown as AddPropertyFormValues['type'],
    addressLine1: '',
    addressLine2: '',
    city: '',
    stateProvince: '',
    postalCode: '',
    country: ORG_DEFAULT_COUNTRY,
    units: 1,
    ownership: '',
    ownerName: '',
    yearBuilt: '',
    onboardDate: iso,
    amenities: [],
    notes: '',
  };
}
