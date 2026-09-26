import { forwardRef, useEffect, useState } from 'react';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogContent from '@mui/material/DialogContent';
import Divider from '@mui/material/Divider';
import IconButton from '@mui/material/IconButton';
import Slide from '@mui/material/Slide';
import type { TransitionProps } from '@mui/material/transitions';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import CloseOutlined from '@mui/icons-material/CloseOutlined';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import { useGenerateRenewalLease } from '../hooks/useLeasesQueries';
import type { LeaseDetailWithUnitProperty } from '../types';

const Transition = forwardRef(function Transition(
  props: TransitionProps & { children: React.ReactElement },
  ref: React.Ref<unknown>,
) {
  return <Slide direction="up" ref={ref} {...props} />;
});

interface GenerateRenewalLeaseDialogProps {
  open: boolean;
  onClose: () => void;
  sourceLease: LeaseDetailWithUnitProperty;
  onSaved: () => void;
}

function addOneYearIso(dateIso: string): string {
  if (!dateIso) return '';
  const d = new Date(dateIso);
  d.setFullYear(d.getFullYear() + 1);
  return d.toISOString().slice(0, 10);
}

/**
 * Turns an Accepted renewal into a new, real Lease record (R7) —
 * pre-filled from the source lease's proposed terms and reviewed here
 * before confirming, rather than silently auto-creating: term/rent/
 * dates are consequential enough to want a final look. The original
 * lease's own record is retired (not rewritten) once this succeeds.
 */
export function GenerateRenewalLeaseDialog({ open, onClose, sourceLease, onSaved }: GenerateRenewalLeaseDialogProps) {
  const generateRenewal = useGenerateRenewalLease(sourceLease.id, sourceLease.unitId);
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [monthlyRent, setMonthlyRent] = useState('');
  const [securityDeposit, setSecurityDeposit] = useState('');
  const [rentDueDay, setRentDueDay] = useState('');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    const start = sourceLease.proposedEndDate || (sourceLease.endDate ? addOneYearIso(sourceLease.endDate) : '');
    setStartDate(sourceLease.proposedEndDate || sourceLease.endDate || '');
    setEndDate(start ? addOneYearIso(start) : '');
    setMonthlyRent(sourceLease.proposedRent != null ? String(sourceLease.proposedRent) : String(sourceLease.monthlyRent));
    setSecurityDeposit(sourceLease.securityDeposit != null ? String(sourceLease.securityDeposit) : '');
    setRentDueDay(sourceLease.rentDueDay != null ? String(sourceLease.rentDueDay) : '');
    setError(null);
  }, [open, sourceLease]);

  const startInvalid = startDate.trim() === '';

  function handleSubmit() {
    if (startInvalid) return;
    setError(null);
    generateRenewal.mutate(
      { startDate, endDate, monthlyRent, securityDeposit, rentDueDay },
      {
        onSuccess: onSaved,
        onError: (err) => setError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.'),
      },
    );
  }

  return (
    <Dialog fullScreen open={open} onClose={onClose} slots={{ transition: Transition }} aria-labelledby="generate-renewal-title">
      <Stack direction="row" sx={{ alignItems: 'flex-start', justifyContent: 'space-between', px: 3, py: 2 }}>
        <Box>
          <Typography id="generate-renewal-title" variant="h4" sx={{ fontSize: 18 }}>
            Generate Renewal Lease
          </Typography>
          <Typography sx={{ fontSize: 12.5, color: tokens.slate[500], mt: 0.25 }}>{sourceLease.primaryResidentName}</Typography>
        </Box>
        <IconButton onClick={onClose} aria-label="Close">
          <CloseOutlined />
        </IconButton>
      </Stack>
      <Divider />

      <DialogContent sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', overflowX: 'hidden' }}>
        <Box sx={{ width: '100%', maxWidth: 480, py: 3 }}>
          <Stack spacing={3}>
            {error && <Alert severity="error">{error}</Alert>}
            <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>
              Review the renewed terms before confirming — this creates a brand-new lease record. The current lease will be retired, not rewritten.
            </Typography>

            <Stack direction="row" spacing={2}>
              <TextField
                label="Start date"
                type="date"
                required
                fullWidth
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                error={startInvalid}
                helperText={startInvalid ? 'Required.' : undefined}
                slotProps={{ inputLabel: { shrink: true } }}
              />
              <TextField
                label="End date"
                type="date"
                fullWidth
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
                placeholder={sourceLease.leaseType === 'month_to_month' ? 'Runs until terminated' : undefined}
                slotProps={{ inputLabel: { shrink: true } }}
              />
            </Stack>
            <Stack direction="row" spacing={2}>
              <TextField
                label="Monthly rent"
                type="number"
                fullWidth
                value={monthlyRent}
                onChange={(e) => setMonthlyRent(e.target.value)}
                slotProps={{ input: { startAdornment: '$' } }}
              />
              <TextField
                label="Security deposit"
                type="number"
                fullWidth
                value={securityDeposit}
                onChange={(e) => setSecurityDeposit(e.target.value)}
                slotProps={{ input: { startAdornment: '$' } }}
              />
            </Stack>
            <TextField label="Rent due day" type="number" value={rentDueDay} onChange={(e) => setRentDueDay(e.target.value)} placeholder="1–31" />
          </Stack>
        </Box>
      </DialogContent>

      <Divider />
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'flex-end', px: 3, py: 2 }}>
        <Stack direction="row" spacing={1.5}>
          <Button variant="outlined" onClick={onClose} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleSubmit}
            disabled={generateRenewal.isPending || startInvalid}
            startIcon={generateRenewal.isPending ? <CircularProgress size={15} color="inherit" /> : undefined}
          >
            {generateRenewal.isPending ? 'Generating…' : 'Generate renewal lease'}
          </Button>
        </Stack>
      </Stack>
    </Dialog>
  );
}
