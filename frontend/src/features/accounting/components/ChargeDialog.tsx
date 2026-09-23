import { useEffect, useState } from 'react';
import { Controller, useForm, useWatch } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import type { PropertyRow } from '@/features/properties/mock/propertyRows';
import { useUnitsPortfolio } from '@/features/units/hooks/useUnitsQueries';
import { useCreateCharge } from '../hooks/useAccountingQueries';
import { chargeDefaultValues, chargeSchema, type ChargeFormValues } from '../schemas/chargeSchema';
import { CHARGE_TYPES, CHARGE_TYPE_LABELS } from '../types';

interface ChargeDialogProps {
  open: boolean;
  onClose: () => void;
  properties: PropertyRow[];
  onSaved: () => void;
}

/**
 * New one-off charge. Picker-first like the other global create flows:
 * choose the property, then the unit; the charge attaches to that unit's
 * active lease (which is what makes it appear in the Rent Roll balance).
 */
export function ChargeDialog({ open, onClose, properties, onSaved }: ChargeDialogProps) {
  const create = useCreateCharge();
  const [propertyId, setPropertyId] = useState('');
  const [submitError, setSubmitError] = useState<string | null>(null);

  const {
    control,
    handleSubmit,
    reset,
    setValue,
    formState: { errors },
  } = useForm<ChargeFormValues>({ resolver: zodResolver(chargeSchema), defaultValues: chargeDefaultValues() });
  const unitId = useWatch({ control, name: 'unitId' });

  const { data: unitsData } = useUnitsPortfolio({ propertyId: propertyId || undefined, limit: 100, offset: 0 });
  const units = propertyId ? (unitsData?.units ?? []) : [];

  useEffect(() => {
    if (open) {
      reset(chargeDefaultValues());
      setPropertyId('');
      setSubmitError(null);
    }
  }, [open, reset]);

  return (
    <Dialog open={open} onClose={create.isPending ? undefined : onClose} fullWidth maxWidth="sm">
      <DialogTitle>New charge</DialogTitle>
      <form
        noValidate
        onSubmit={handleSubmit((values) => {
          setSubmitError(null);
          create.mutate(values, {
            onSuccess: onSaved,
            onError: (err) => setSubmitError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.'),
          });
        })}
      >
        <DialogContent>
          <Stack spacing={2}>
            <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>
              Bill a tenant for something beyond rent. It is added to their balance on the Rent Roll for the month it is due.
              Late fees are added automatically and can&apos;t be created here.
            </Typography>
            {submitError && <Alert severity="error">{submitError}</Alert>}

            <TextField
              select
              label="Property"
              value={propertyId}
              onChange={(e) => {
                setPropertyId(e.target.value);
                setValue('unitId', '');
              }}
              required
            >
              {properties.map((p) => (
                <MenuItem key={p.id} value={p.id}>
                  {p.name}
                </MenuItem>
              ))}
            </TextField>
            <Controller
              name="unitId"
              control={control}
              render={({ field }) => (
                <TextField
                  {...field}
                  select
                  required
                  label="Unit"
                  disabled={!propertyId}
                  error={Boolean(errors.unitId)}
                  helperText={errors.unitId?.message}
                >
                  {units.map((u) => (
                    <MenuItem key={u.id} value={u.id}>
                      {u.unitName}
                    </MenuItem>
                  ))}
                </TextField>
              )}
            />
            <Controller
              name="chargeType"
              control={control}
              render={({ field }) => (
                <TextField
                  {...field}
                  value={field.value ?? ''}
                  select
                  required
                  label="Charge type"
                  error={Boolean(errors.chargeType)}
                  helperText={errors.chargeType?.message}
                >
                  {CHARGE_TYPES.map((t) => (
                    <MenuItem key={t} value={t}>
                      {CHARGE_TYPE_LABELS[t]}
                    </MenuItem>
                  ))}
                </TextField>
              )}
            />
            <Stack direction="row" spacing={2}>
              <Controller
                name="amount"
                control={control}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Amount"
                    type="number"
                    required
                    sx={{ flex: 1 }}
                    error={Boolean(errors.amount)}
                    helperText={errors.amount?.message}
                    slotProps={{ input: { startAdornment: '$' }, htmlInput: { step: '0.01', min: 0 } }}
                  />
                )}
              />
              <Controller
                name="dueOn"
                control={control}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Due date"
                    type="date"
                    required
                    sx={{ flex: 1 }}
                    error={Boolean(errors.dueOn)}
                    helperText={errors.dueOn?.message}
                    slotProps={{ inputLabel: { shrink: true } }}
                  />
                )}
              />
            </Stack>
            <Controller
              name="description"
              control={control}
              render={({ field }) => (
                <TextField
                  {...field}
                  label="Description"
                  required
                  multiline
                  minRows={2}
                  error={Boolean(errors.description)}
                  helperText={errors.description?.message ?? 'Shown to the tenant, e.g. "Water — August"'}
                />
              )}
            />
          </Stack>
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 0 }}>
          <Button variant="text" onClick={onClose} disabled={create.isPending} sx={{ color: tokens.slate[600] }}>
            Cancel
          </Button>
          <Button
            type="submit"
            variant="contained"
            disabled={create.isPending || !unitId}
            startIcon={create.isPending ? <CircularProgress size={15} color="inherit" /> : undefined}
          >
            {create.isPending ? 'Saving…' : 'Create charge'}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
}
