import { useRef } from 'react';
import Autocomplete from '@mui/material/Autocomplete';
import TextField from '@mui/material/TextField';
import { Controller, type Control, type UseFormSetValue } from 'react-hook-form';
import { MOCK_ADDRESS_SUGGESTIONS, parseFreeformAddress, type AddressSuggestion } from '../../mock/addressSuggestions';
import type { AddPropertyFormValues } from '../../schemas/addPropertySchema';

interface AddressAutocompleteFieldProps {
  control: Control<AddPropertyFormValues>;
  setValue: UseFormSetValue<AddPropertyFormValues>;
  error?: string;
  onBlurExtra: () => void;
}

/**
 * A single address input with suggestions — stands in for a real
 * geocoding autocomplete (Google Places, Mapbox, ...), which needs an
 * API key and a backend proxy that don't exist yet (see
 * mock/addressSuggestions.ts). Picking a suggestion, or just typing a
 * well-formed address and tabbing away, fills addressLine1/city/state/
 * postal/country — all of which stay independently editable afterward
 * via BasicInfoStep's fields below this one.
 */
export function AddressAutocompleteField({ control, setValue, error, onBlurExtra }: AddressAutocompleteFieldProps) {
  // Tracks whether the structured fields were just set from a picked
  // suggestion, so onBlur's freeform-parse fallback (for someone who
  // typed their own address instead) doesn't immediately overwrite
  // them. Without this, picking "214 Willow Creek Rd, Austin, TX
  // 78745..." correctly set stateProvince to "Texas" via onChange, but
  // the blur that follows every selection re-parsed that same fullAddress
  // string — which itself reads "TX", the standard postal abbreviation,
  // not "Texas" — and clobbered it right back to "TX". That's a valid
  // value on its own, but not one that matches any option in the State
  // dropdown's full-name list, so the field then displays as blank.
  const suggestionJustPicked = useRef(false);

  return (
    <Controller
      name="fullAddress"
      control={control}
      render={({ field }) => (
        <Autocomplete<AddressSuggestion, false, false, true>
          freeSolo
          options={MOCK_ADDRESS_SUGGESTIONS}
          getOptionLabel={(opt) => (typeof opt === 'string' ? opt : opt.fullAddress)}
          inputValue={field.value}
          onInputChange={(_e, value, reason) => {
            field.onChange(value);
            // 'input' is real typing; MUI also fires this with reason
            // 'reset' right after onChange, to sync the input's text to
            // the picked option's label — that one must NOT clear the
            // flag onChange just set.
            if (reason === 'input') suggestionJustPicked.current = false;
          }}
          onChange={(_e, value) => {
            if (value && typeof value !== 'string') {
              field.onChange(value.fullAddress);
              setValue('addressLine1', value.addressLine1, { shouldDirty: true });
              setValue('city', value.city, { shouldDirty: true });
              setValue('stateProvince', value.stateProvince, { shouldDirty: true });
              setValue('postalCode', value.postalCode, { shouldDirty: true });
              setValue('country', value.country, { shouldDirty: true });
              suggestionJustPicked.current = true;
            }
          }}
          onBlur={() => {
            field.onBlur();
            if (!suggestionJustPicked.current) {
              const parsed = parseFreeformAddress(field.value);
              if (parsed.addressLine1) setValue('addressLine1', parsed.addressLine1, { shouldDirty: true });
              if (parsed.city) setValue('city', parsed.city, { shouldDirty: true });
              if (parsed.stateProvince) setValue('stateProvince', parsed.stateProvince, { shouldDirty: true });
              if (parsed.postalCode) setValue('postalCode', parsed.postalCode, { shouldDirty: true });
              if (parsed.country) setValue('country', parsed.country, { shouldDirty: true });
            }
            onBlurExtra();
          }}
          renderInput={(params) => (
            <TextField
              {...params}
              id="field-fullAddress"
              label="Address"
              required
              placeholder="Start typing an address…"
              error={Boolean(error)}
              helperText={
                error ??
                'Pick a suggestion to auto-fill city, state, and postal code — or type your own and edit the fields below.'
              }
            />
          )}
        />
      )}
    />
  );
}
