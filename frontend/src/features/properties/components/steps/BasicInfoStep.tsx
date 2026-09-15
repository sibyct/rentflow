import { Controller, type Control, type FieldErrors, type UseFormSetValue, type UseFormTrigger } from 'react-hook-form';
import Box from '@mui/material/Box';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import Typography from '@mui/material/Typography';
import { tokens } from '@/app/tokens';
import { AddressAutocompleteField } from './AddressAutocompleteField';
import { PROPERTY_TYPES, US_STATES, COUNTRIES } from '../../mock/propertyRows';
import type { AddPropertyFormValues } from '../../schemas/addPropertySchema';

interface BasicInfoStepProps {
  control: Control<AddPropertyFormValues>;
  setValue: UseFormSetValue<AddPropertyFormValues>;
  trigger: UseFormTrigger<AddPropertyFormValues>;
  errors: FieldErrors<AddPropertyFormValues>;
  touched: Partial<Record<keyof AddPropertyFormValues, boolean>>;
  markTouched: (key: keyof AddPropertyFormValues) => void;
}

export function BasicInfoStep({ control, setValue, trigger, errors, touched, markTouched }: BasicInfoStepProps) {
  return (
    <Stack spacing={2.5}>
      <Controller
        name="name"
        control={control}
        render={({ field }) => (
          <TextField
            {...field}
            id="field-name"
            label="Property name"
            required
            fullWidth
            onBlur={() => {
              field.onBlur();
              markTouched('name');
              void trigger('name');
            }}
            error={Boolean(touched.name && errors.name)}
            helperText={touched.name && errors.name?.message}
            placeholder="e.g. Willow Creek Apartments"
          />
        )}
      />

      <Box id="field-type">
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: 'text.secondary', mb: 1 }}>
          Property type{' '}
          <Box component="span" sx={{ color: 'error.main' }}>
            *
          </Box>
        </Typography>
        <Controller
          name="type"
          control={control}
          render={({ field }) => (
            <ToggleButtonGroup
              exclusive
              value={field.value ?? null}
              onChange={(_e, value: AddPropertyFormValues['type'] | null) => {
                if (!value) return;
                field.onChange(value);
                markTouched('type');
                void trigger('type');
                if (value === 'Residential – Single Unit') setValue('units', 1, { shouldValidate: true });
              }}
              sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 1.5, width: '100%' }}
            >
              {PROPERTY_TYPES.map((t) => (
                <ToggleButton
                  key={t}
                  value={t}
                  sx={{
                    border: `1.5px solid ${tokens.slate[200]} !important`,
                    borderRadius: `${tokens.radiusCard}px !important`,
                    textTransform: 'none',
                    fontWeight: 600,
                    fontSize: 13.5,
                    color: tokens.slate[600],
                    justifyContent: 'flex-start',
                    textAlign: 'left',
                    lineHeight: 1.3,
                    py: 1.75,
                    px: 2,
                    '&.Mui-selected': {
                      bgcolor: tokens.azure[50],
                      color: tokens.azure[700],
                      borderColor: `${tokens.azure[500]} !important`,
                    },
                    '&.Mui-selected:hover': { bgcolor: tokens.azure[50] },
                  }}
                >
                  {t.replace(' – ', ' –​ ')}
                </ToggleButton>
              ))}
            </ToggleButtonGroup>
          )}
        />
        {touched.type && errors.type && (
          <Typography sx={{ fontSize: 12, color: 'error.main', mt: 0.75 }}>{errors.type.message}</Typography>
        )}
      </Box>

      <AddressAutocompleteField
        control={control}
        setValue={setValue}
        error={touched.fullAddress ? errors.fullAddress?.message : undefined}
        onBlurExtra={() => {
          markTouched('fullAddress');
          void trigger('fullAddress');
        }}
      />

      {/* Auto-filled by the address field above, and independently
          editable — nothing here is required (see addPropertySchema). */}
      <Stack spacing={2}>
        <Controller
          name="addressLine1"
          control={control}
          render={({ field }) => <TextField {...field} label="Street address" fullWidth />}
        />
        <Controller
          name="addressLine2"
          control={control}
          render={({ field }) => <TextField {...field} label="Apt / suite" fullWidth placeholder="Unit, suite, floor (optional)" />}
        />
        <Stack direction="row" spacing={2} sx={{ flexWrap: 'wrap', rowGap: 2 }}>
          <Controller
            name="city"
            control={control}
            render={({ field }) => <TextField {...field} label="City" sx={{ flex: '1 1 160px' }} />}
          />
          <Controller
            name="stateProvince"
            control={control}
            render={({ field }) => (
              <TextField {...field} select label="State / province" sx={{ flex: '1 1 160px' }}>
                <MenuItem value="">
                  <em>None</em>
                </MenuItem>
                {US_STATES.map((s) => (
                  <MenuItem key={s} value={s}>
                    {s}
                  </MenuItem>
                ))}
              </TextField>
            )}
          />
        </Stack>
        <Stack direction="row" spacing={2} sx={{ flexWrap: 'wrap', rowGap: 2 }}>
          <Controller
            name="postalCode"
            control={control}
            render={({ field }) => <TextField {...field} label="Postal code" sx={{ flex: '1 1 160px' }} />}
          />
          <Controller
            name="country"
            control={control}
            render={({ field }) => (
              <TextField {...field} select label="Country" sx={{ flex: '1 1 160px' }}>
                {COUNTRIES.map((c) => (
                  <MenuItem key={c} value={c}>
                    {c}
                  </MenuItem>
                ))}
              </TextField>
            )}
          />
        </Stack>
      </Stack>
    </Stack>
  );
}
