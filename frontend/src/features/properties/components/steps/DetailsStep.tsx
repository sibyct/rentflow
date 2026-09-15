import dayjs from 'dayjs';
import { Controller, type Control, type FieldErrors, type UseFormTrigger } from 'react-hook-form';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDayjs } from '@mui/x-date-pickers/AdapterDayjs';
import { DatePicker } from '@mui/x-date-pickers/DatePicker';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Collapse from '@mui/material/Collapse';
import FormControlLabel from '@mui/material/FormControlLabel';
import IconButton from '@mui/material/IconButton';
import Radio from '@mui/material/Radio';
import RadioGroup from '@mui/material/RadioGroup';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import AddOutlined from '@mui/icons-material/AddOutlined';
import RemoveOutlined from '@mui/icons-material/RemoveOutlined';
import { tokens } from '@/app/tokens';
import type { AddPropertyFormValues } from '../../schemas/addPropertySchema';

interface DetailsStepProps {
  control: Control<AddPropertyFormValues>;
  trigger: UseFormTrigger<AddPropertyFormValues>;
  errors: FieldErrors<AddPropertyFormValues>;
  touched: Partial<Record<keyof AddPropertyFormValues, boolean>>;
  markTouched: (key: keyof AddPropertyFormValues) => void;
  unitsLocked: boolean;
  ownership: AddPropertyFormValues['ownership'];
}

const ISO_FORMAT = 'YYYY-MM-DD';

export function DetailsStep({ control, trigger, errors, touched, markTouched, unitsLocked, ownership }: DetailsStepProps) {
  return (
    <Stack spacing={2.5}>
      <Box>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: 'text.secondary', mb: 1 }}>
          Number of units{' '}
          <Box component="span" sx={{ color: 'error.main' }}>
            *
          </Box>
        </Typography>
        <Controller
          name="units"
          control={control}
          render={({ field }) => (
            <Tooltip title={unitsLocked ? 'Single-unit properties always have 1 unit.' : ''} placement="right">
              <span style={{ display: 'inline-block', width: 160 }}>
                <TextField
                  id="field-units"
                  value={field.value}
                  disabled={unitsLocked}
                  onChange={(e) => {
                    const n = parseInt(e.target.value.replace(/\D/g, ''), 10);
                    field.onChange(Number.isNaN(n) ? 0 : n);
                  }}
                  onBlur={() => {
                    if (!field.value || field.value < 1) field.onChange(1);
                    field.onBlur();
                    markTouched('units');
                    void trigger('units');
                  }}
                  error={Boolean(touched.units && errors.units)}
                  helperText={touched.units && errors.units?.message}
                  slotProps={{
                    input: {
                      startAdornment: (
                        <IconButton
                          size="small"
                          disabled={unitsLocked}
                          onClick={() => field.onChange(Math.max(1, (field.value || 1) - 1))}
                          aria-label="Decrease units"
                        >
                          <RemoveOutlined fontSize="small" />
                        </IconButton>
                      ),
                      endAdornment: (
                        <IconButton
                          size="small"
                          disabled={unitsLocked}
                          onClick={() => field.onChange((field.value || 1) + 1)}
                          aria-label="Increase units"
                        >
                          <AddOutlined fontSize="small" />
                        </IconButton>
                      ),
                      sx: { pl: 0.5, pr: 0.5 },
                    },
                    htmlInput: { style: { textAlign: 'center', fontFamily: tokens.fontMono, fontWeight: 600 } },
                  }}
                  fullWidth
                />
              </span>
            </Tooltip>
          )}
        />
      </Box>

      <Box>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: 'text.secondary', mb: 1 }}>
          Ownership type{' '}
          <Box component="span" sx={{ color: tokens.slate[400], fontWeight: 500 }}>
            (optional)
          </Box>
        </Typography>
        <Controller
          name="ownership"
          control={control}
          render={({ field }) => (
            <RadioGroup row {...field} value={field.value ?? ''}>
              <FormControlLabel value="owned" control={<Radio size="small" />} label="Owned" />
              <FormControlLabel value="managed" control={<Radio size="small" />} label="Managed on behalf of owner" />
            </RadioGroup>
          )}
        />
      </Box>

      <Collapse in={ownership === 'managed'}>
        <Controller
          name="ownerName"
          control={control}
          render={({ field }) => (
            <TextField {...field} label="Owner name / entity" fullWidth placeholder="e.g. Alvarez Family Trust" />
          )}
        />
      </Collapse>

      <Stack direction="row" spacing={2} sx={{ flexWrap: 'wrap', rowGap: 2.5 }}>
        <Controller
          name="yearBuilt"
          control={control}
          render={({ field }) => (
            <TextField
              {...field}
              label="Year built"
              sx={{ flex: '1 1 160px' }}
              placeholder="(optional)"
              slotProps={{ htmlInput: { inputMode: 'numeric', maxLength: 4, style: { fontFamily: tokens.fontMono } } }}
            />
          )}
        />

        <Box sx={{ flex: '1 1 220px' }}>
          <LocalizationProvider dateAdapter={AdapterDayjs}>
            <Controller
              name="onboardDate"
              control={control}
              render={({ field }) => (
                <Stack spacing={0.75}>
                  <DatePicker
                    label="Onboard date"
                    value={field.value ? dayjs(field.value) : null}
                    onChange={(newValue) => field.onChange(newValue?.isValid() ? newValue.format(ISO_FORMAT) : '')}
                    slotProps={{ textField: { fullWidth: true } }}
                  />
                  <Box>
                    <Chip
                      label="Today"
                      size="small"
                      onClick={() => field.onChange(dayjs().format(ISO_FORMAT))}
                      variant={field.value === dayjs().format(ISO_FORMAT) ? 'filled' : 'outlined'}
                      color={field.value === dayjs().format(ISO_FORMAT) ? 'primary' : 'default'}
                    />
                  </Box>
                </Stack>
              )}
            />
          </LocalizationProvider>
        </Box>
      </Stack>
    </Stack>
  );
}
