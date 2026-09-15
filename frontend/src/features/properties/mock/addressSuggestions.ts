// Stand-in for a real address-autocomplete/geocoding service (Google
// Places, Mapbox, Smarty, etc.). A real integration needs an API key and
// should be proxied through the backend — never call a geocoding API
// with a key embedded in client code — which doesn't exist yet. This
// gives the same *interaction* (type, see suggestions, pick one, fields
// auto-fill and stay editable) against a small local list, following the
// same mock-data convention as mock/propertyRows.ts.

export interface AddressSuggestion {
  fullAddress: string;
  addressLine1: string;
  city: string;
  stateProvince: string;
  postalCode: string;
  country: string;
}

export const MOCK_ADDRESS_SUGGESTIONS: AddressSuggestion[] = [
  {
    fullAddress: '214 Willow Creek Rd, Austin, TX 78745, United States',
    addressLine1: '214 Willow Creek Rd',
    city: 'Austin',
    stateProvince: 'Texas',
    postalCode: '78745',
    country: 'United States',
  },
  {
    fullAddress: '88 Hendricks Ave, Austin, TX 78702, United States',
    addressLine1: '88 Hendricks Ave',
    city: 'Austin',
    stateProvince: 'Texas',
    postalCode: '78702',
    country: 'United States',
  },
  {
    fullAddress: '902 Maple St, Austin, TX 78703, United States',
    addressLine1: '902 Maple St',
    city: 'Austin',
    stateProvince: 'Texas',
    postalCode: '78703',
    country: 'United States',
  },
  {
    fullAddress: '47 Sable Point Ln, Round Rock, TX 78664, United States',
    addressLine1: '47 Sable Point Ln',
    city: 'Round Rock',
    stateProvince: 'Texas',
    postalCode: '78664',
    country: 'United States',
  },
  {
    fullAddress: '500 Riverside Dr, Austin, TX 78741, United States',
    addressLine1: '500 Riverside Dr',
    city: 'Austin',
    stateProvince: 'Texas',
    postalCode: '78741',
    country: 'United States',
  },
  {
    fullAddress: '1600 Pennsylvania Ave NW, Washington, DC 20500, United States',
    addressLine1: '1600 Pennsylvania Ave NW',
    city: 'Washington',
    stateProvince: 'District of Columbia',
    postalCode: '20500',
    country: 'United States',
  },
  {
    fullAddress: '350 5th Ave, New York, NY 10118, United States',
    addressLine1: '350 5th Ave',
    city: 'New York',
    stateProvince: 'New York',
    postalCode: '10118',
    country: 'United States',
  },
];

/**
 * Best-effort split of a freely-typed address into parts, for when
 * someone types a full address rather than picking a suggestion.
 * Handles "street, city, ST zip" and "street, city, state, zip[, country]"
 * — anything else just leaves the structured fields blank for manual
 * entry, which is fine since fullAddress alone is what's required.
 */
export function parseFreeformAddress(input: string): Partial<AddressSuggestion> {
  const parts = input
    .split(',')
    .map((p) => p.trim())
    .filter(Boolean);
  if (parts.length < 2) return {};

  const [addressLine1, city, ...rest] = parts;
  const result: Partial<AddressSuggestion> = { addressLine1, city };

  if (rest.length === 0) return result;

  // "TX 78701" (state abbreviation + zip) vs "Texas" / "78701" as separate parts
  const stateZipMatch = rest[0]?.match(/^([A-Za-z ]+?)\s+(\d{5}(?:-\d{4})?)$/);
  if (stateZipMatch) {
    result.stateProvince = stateZipMatch[1];
    result.postalCode = stateZipMatch[2];
    if (rest[1]) result.country = rest[1];
    return result;
  }

  if (rest[0]) result.stateProvince = rest[0];
  if (rest[1]) result.postalCode = rest[1];
  if (rest[2]) result.country = rest[2];
  return result;
}
