import { zodResolver } from '@hookform/resolvers/zod';
import { Controller, useForm } from 'react-hook-form';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import type { CreatePropertyInput, PropertyStatus } from '../types';
import { createPropertySchema, type CreatePropertyFormValues } from '../schemas/propertySchema';

interface PropertyFormProps {
  onSubmit: (input: CreatePropertyInput) => void;
  isSubmitting?: boolean;
  submitError?: unknown;
}

const STATUS_OPTIONS: PropertyStatus[] = ['active', 'inactive', 'maintenance'];

export function PropertyForm({ onSubmit, isSubmitting, submitError }: PropertyFormProps) {
  const {
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<CreatePropertyFormValues>({
    resolver: zodResolver(createPropertySchema),
    mode: 'onTouched',
    defaultValues: { address: '', unitCount: 1, status: 'active' },
  });

  const onValid = (values: CreatePropertyFormValues) => {
    onSubmit(values);
  };

  return (
    <Box component="form" onSubmit={handleSubmit(onValid)} noValidate sx={{ maxWidth: 420 }}>
      <Stack spacing={2}>
        {/* Controller, not register: MUI's inputs are controlled
            components — Controller drives them via explicit value/onChange
            rather than the ref-based uncontrolled wiring register() uses,
            which is also what MUI's own docs recommend for 3rd-party form
            libraries. */}
        <Controller
          name="address"
          control={control}
          render={({ field }) => (
            <TextField
              {...field}
              label="Address"
              fullWidth
              error={Boolean(errors.address)}
              helperText={errors.address?.message}
            />
          )}
        />

        <Controller
          name="unitCount"
          control={control}
          render={({ field }) => (
            <TextField
              {...field}
              label="Unit count"
              type="number"
              fullWidth
              onChange={(e) => field.onChange(Number(e.target.value))}
              error={Boolean(errors.unitCount)}
              helperText={errors.unitCount?.message}
            />
          )}
        />

        <Controller
          name="status"
          control={control}
          render={({ field }) => (
            <TextField
              {...field}
              select
              label="Status"
              fullWidth
              error={Boolean(errors.status)}
              helperText={errors.status?.message}
            >
              {STATUS_OPTIONS.map((option) => (
                <MenuItem key={option} value={option}>
                  {option}
                </MenuItem>
              ))}
            </TextField>
          )}
        />

        {submitError !== undefined && submitError !== null && (
          <Alert severity="error">Could not save the property. Please try again.</Alert>
        )}

        <Button type="submit" variant="contained" disabled={isSubmitting}>
          {isSubmitting ? 'Saving…' : 'Create property'}
        </Button>
      </Stack>
    </Box>
  );
}
