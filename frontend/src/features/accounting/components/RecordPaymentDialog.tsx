import { useEffect } from 'react';
import { Controller, useForm } from 'react-hook-form';
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
import { tokens } from '@/app/tokens';
import { dollarsToCents, formatMoney } from '@/shared/lib/format';
import { paymentDefaultValues, paymentSchema, type PaymentFormValues } from '../schemas/paymentSchema';
import { PAYMENT_METHODS } from '../types';

interface RecordPaymentDialogProps {
  open: boolean;
  title: string;
  /** e.g. "Sunrise Apts · Unit 4 — Jane Doe" */
  subtitle?: string;
  outstandingCents: number;
  submitting: boolean;
  error?: string | null;
  onClose: () => void;
  onSubmit: (values: PaymentFormValues) => void;
}

/**
 * One payment form for both places money gets recorded: a lump sum from
 * the Rent Roll (allocated across the lease's open rows server-side) and
 * a payment on a single expense. The caller supplies the mutation.
 */
export function RecordPaymentDialog({
  open,
  title,
  subtitle,
  outstandingCents,
  submitting,
  error,
  onClose,
  onSubmit,
}: RecordPaymentDialogProps) {
  const {
    control,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<PaymentFormValues>({
    resolver: zodResolver(
      paymentSchema.refine((v) => (dollarsToCents(v.amount) ?? 0) <= outstandingCents, {
        path: ['amount'],
        message: `Cannot exceed the ${formatMoney(outstandingCents)} still owed.`,
      }),
    ),
    defaultValues: paymentDefaultValues(outstandingCents),
  });

  useEffect(() => {
    if (open) reset(paymentDefaultValues(outstandingCents));
  }, [open, outstandingCents, reset]);

  return (
    <Dialog open={open} onClose={submitting ? undefined : onClose} fullWidth maxWidth="xs">
      <DialogTitle>{title}</DialogTitle>
      <form onSubmit={handleSubmit(onSubmit)} noValidate>
        <DialogContent>
          <Stack spacing={2}>
            {subtitle && <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>{subtitle}</Typography>}
            <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>
              Outstanding: <strong>{formatMoney(outstandingCents)}</strong>
            </Typography>
            {error && <Alert severity="error">{error}</Alert>}

            <Controller
              name="amount"
              control={control}
              render={({ field }) => (
                <TextField
                  {...field}
                  label="Amount"
                  type="number"
                  required
                  autoFocus
                  error={Boolean(errors.amount)}
                  helperText={errors.amount?.message}
                  slotProps={{ input: { startAdornment: '$' }, htmlInput: { step: '0.01', min: 0 } }}
                />
              )}
            />
            <Controller
              name="paidOn"
              control={control}
              render={({ field }) => (
                <TextField
                  {...field}
                  label="Date received"
                  type="date"
                  required
                  error={Boolean(errors.paidOn)}
                  helperText={errors.paidOn?.message}
                  slotProps={{ inputLabel: { shrink: true } }}
                />
              )}
            />
            <Stack direction="row" spacing={2}>
              <Controller
                name="method"
                control={control}
                render={({ field }) => (
                  <TextField {...field} select label="Method" sx={{ flex: 1 }} slotProps={{ inputLabel: { shrink: true }, select: { displayEmpty: true } }}>
                    <MenuItem value="">—</MenuItem>
                    {PAYMENT_METHODS.map((m) => (
                      <MenuItem key={m} value={m}>
                        {m}
                      </MenuItem>
                    ))}
                  </TextField>
                )}
              />
              <Controller
                name="reference"
                control={control}
                render={({ field }) => (
                  <TextField {...field} label="Reference" placeholder="Check #, txn id" sx={{ flex: 1 }} error={Boolean(errors.reference)} />
                )}
              />
            </Stack>
          </Stack>
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 0 }}>
          <Button variant="text" onClick={onClose} disabled={submitting} sx={{ color: tokens.slate[600] }}>
            Cancel
          </Button>
          <Button type="submit" variant="contained" disabled={submitting} startIcon={submitting ? <CircularProgress size={15} color="inherit" /> : undefined}>
            {submitting ? 'Saving…' : 'Record payment'}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
}
