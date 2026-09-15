import { Controller, type Control } from 'react-hook-form';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import type { AddPropertyFormValues } from '../../schemas/addPropertySchema';

interface NotesStepProps {
  control: Control<AddPropertyFormValues>;
}

export function NotesStep({ control }: NotesStepProps) {
  return (
    <Stack spacing={1}>
      <Controller
        name="notes"
        control={control}
        render={({ field }) => (
          <TextField
            {...field}
            multiline
            minRows={6}
            fullWidth
            label="Internal notes"
            // A persistent caption, not just placeholder ghost text — it
            // stays visible once the field has content, unlike a
            // placeholder which disappears on the first keystroke.
            helperText="Internal notes — not visible to tenants."
          />
        )}
      />
    </Stack>
  );
}
