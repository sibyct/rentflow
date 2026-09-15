import { z } from 'zod';
import { PROPERTY_TYPES, ORG_DEFAULT_COUNTRY } from '../mock/propertyRows';

// Client-side-only form: the fields here (type, city/state/postal,
// ownership, amenities, ...) aren't part of the backend's Property DTO
// yet — see mock/propertyRows.ts for why this modal works against local
// state instead of CreatePropertyInput/createPropertySchema.
//
// Only name, type, fullAddress, and units are required — see
// AddPropertyModal's stepper: those are the fields that block "Next".
// addressLine1/city/stateProvince/postalCode/country are filled by the
// address autocomplete (see mock/addressSuggestions.ts) but stay
// optional and independently editable, since the single fullAddress
// field is what's actually required.
export const addPropertySchema = z.object({
  // Step 1 — Basic Info
  name: z.string().trim().min(1, 'Property name is required.'),
  type: z.enum(PROPERTY_TYPES, { error: 'Select a property type.' }),
  fullAddress: z.string().trim().min(1, 'Address is required.'),
  addressLine1: z.string().trim().optional(),
  addressLine2: z.string().trim().optional(),
  city: z.string().trim().optional(),
  stateProvince: z.string().trim().optional(),
  postalCode: z.string().trim().optional(),
  country: z.string().trim().optional(),

  // Step 2 — Details
  units: z
    .number({ error: 'Enter at least 1 unit.' })
    .int('Enter a whole number of units.')
    .min(1, 'Enter at least 1 unit.'),
  ownership: z.union([z.enum(['owned', 'managed']), z.literal('')]).optional(),
  ownerName: z.string().trim().optional(),
  yearBuilt: z.string().trim().optional(),
  /** ISO 'YYYY-MM-DD', or '' when unset — blank by default (see AddPropertyModal's "Today" chip). */
  onboardDate: z.string().trim().optional(),

  // Step 3 — Amenities & Media
  // No .default() here: RHF's own defaultValues (addPropertyDefaultValues
  // below) already seeds this as []. A zod .default() makes the schema's
  // input/output types diverge (input allows undefined, output doesn't),
  // which is what was causing @hookform/resolvers' Resolver<> type to
  // stop lining up with useForm<AddPropertyFormValues> — the "Two
  // different types with this name exist" error seen throughout this
  // file and (before this fix) AddPropertyDrawer.
  amenities: z.array(z.string()),

  // Step 4 — Notes
  notes: z.string().trim().optional(),
});

export type AddPropertyFormValues = z.infer<typeof addPropertySchema>;

// Per-step required-field subsets, used to gate each step's "Next"
// button independently of the full form's validity (see
// AddPropertyModal's useStepValidity).
export const stepSchemas = {
  basics: addPropertySchema.pick({ name: true, type: true, fullAddress: true }),
  details: addPropertySchema.pick({ units: true }),
  amenities: addPropertySchema.pick({}),
  notes: addPropertySchema.pick({}),
} as const;

export function addPropertyDefaultValues(): AddPropertyFormValues {
  return {
    name: '',
    type: undefined as unknown as AddPropertyFormValues['type'],
    fullAddress: '',
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
    onboardDate: '',
    amenities: [],
    notes: '',
  };
}
