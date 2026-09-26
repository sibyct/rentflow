import { useEffect, useState } from 'react';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import FormControlLabel from '@mui/material/FormControlLabel';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import { useChangeRent } from '../hooks/useLeasesQueries';

interface ChangeRentDialogProps {
  open: boolean;
  onClose: () => void;
  leaseId: string;
  unitId: string;
  currentAmount: number;
  onSaved: () => void;
}

function todayIso(): string {
  return new Date().toISOString().slice(0, 10);
}

/**
 * The only way to change an active lease's effective rent (R3/R4/R6) —
 * a lightweight, non-fullscreen Dialog (three fields, no need for the
 * fullscreen Slide chrome the bigger Terminate/Generate Renewal
 * dialogs use) that always appends a dated amendment, never overwrites
 * the lease's rent directly.
 */
export function ChangeRentDialog({ open, onClose, leaseId, unitId, currentAmount, onSaved }: ChangeRentDialogProps) {
  const changeRent = useChangeRent(leaseId, unitId);
  const [amount, setAmount] = useState('');
  const [effectiveDate, setEffectiveDate] = useState('');
  const [reason, setReason] = useState('');
  const [isCorrection, setIsCorrection] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setAmount(String(currentAmount));
    setEffectiveDate(todayIso());
    setReason('');
    setIsCorrection(false);
    setError(null);
  }, [open, currentAmount]);

  const amountInvalid = amount.trim() === '' || Number(amount) < 0;
  const dateInvalid = effectiveDate.trim() === '';
  // A rent change is never backdated (R6) except as an explicitly
  // flagged data-entry correction.
  const isPastDate = effectiveDate !== '' && effectiveDate < todayIso();

  function handleSubmit() {
    if (amountInvalid || dateInvalid || (isPastDate && !isCorrection)) return;
    setError(null);
    changeRent.mutate(
      { amount, effectiveDate, reason, isCorrection },
      {
        onSuccess: onSaved,
        onError: (err) => setError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.'),
      },
    );
  }

  return (
    <Dialog open={open} onClose={changeRent.isPending ? undefined : onClose} fullWidth maxWidth="xs">
      <DialogTitle>Change rent</DialogTitle>
      <DialogContent>
        <Stack spacing={2}>
          <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>
            Current rent: <strong>${currentAmount.toLocaleString()}</strong>. This records a dated amendment — it never overwrites the lease's rent directly.
          </Typography>
          {error && <Alert severity="error">{error}</Alert>}

          <TextField
            label="New amount"
            type="number"
            required
            autoFocus
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            error={amountInvalid}
            helperText={amountInvalid ? 'Enter a non-negative amount.' : undefined}
            slotProps={{ input: { startAdornment: '$' } }}
          />
          <TextField
            label="Effective date"
            type="date"
            required
            value={effectiveDate}
            onChange={(e) => setEffectiveDate(e.target.value)}
            error={dateInvalid || (isPastDate && !isCorrection)}
            helperText={
              isPastDate && !isCorrection
                ? 'A past date is only allowed as a flagged correction below.'
                : 'Must be today or later, unless flagged as a correction.'
            }
            slotProps={{ inputLabel: { shrink: true } }}
          />
          <TextField label="Reason" value={reason} onChange={(e) => setReason(e.target.value)} placeholder="e.g. annual increase" fullWidth />
          <FormControlLabel
            control={<Checkbox checked={isCorrection} onChange={(e) => setIsCorrection(e.target.checked)} />}
            label="This is a data-entry correction (allows a past effective date)"
          />
        </Stack>
      </DialogContent>
      <DialogActions sx={{ p: 2.5, pt: 0 }}>
        <Button variant="text" onClick={onClose} disabled={changeRent.isPending} sx={{ color: tokens.slate[600] }}>
          Cancel
        </Button>
        <Button
          variant="contained"
          onClick={handleSubmit}
          disabled={changeRent.isPending || amountInvalid || dateInvalid || (isPastDate && !isCorrection)}
          startIcon={changeRent.isPending ? <CircularProgress size={15} color="inherit" /> : undefined}
        >
          {changeRent.isPending ? 'Saving…' : 'Change rent'}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
