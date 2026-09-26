import { forwardRef, useEffect, useState } from 'react';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogContent from '@mui/material/DialogContent';
import Divider from '@mui/material/Divider';
import IconButton from '@mui/material/IconButton';
import MenuItem from '@mui/material/MenuItem';
import Slide from '@mui/material/Slide';
import type { TransitionProps } from '@mui/material/transitions';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import CloseOutlined from '@mui/icons-material/CloseOutlined';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import { useTerminateLease } from '../hooks/useLeasesQueries';
import { TERMINATION_REASON_LABELS, TERMINATION_REASON_OPTIONS, type TerminationReason } from '../types';

const Transition = forwardRef(function Transition(
  props: TransitionProps & { children: React.ReactElement },
  ref: React.Ref<unknown>,
) {
  return <Slide direction="up" ref={ref} {...props} />;
});

interface TerminateLeaseDialogProps {
  open: boolean;
  onClose: () => void;
  leaseId: string;
  unitId: string;
  residentName: string;
  onSaved: () => void;
}

/**
 * The dedicated action for ending a lease (R10) — deliberately not a
 * generic status edit through LeaseFormDialog, so the unit-vacancy
 * wiring and audit entry always happen together. Fullscreen Slide
 * chrome matching LeaseFormDialog/UnitFormDialog, since this is a
 * consequential, form-like action (termination reason, notice date,
 * optional move-out inspection), not a quick three-field change like
 * ChangeRentDialog.
 */
export function TerminateLeaseDialog({ open, onClose, leaseId, unitId, residentName, onSaved }: TerminateLeaseDialogProps) {
  const terminateLease = useTerminateLease(leaseId, unitId);
  const [terminationReason, setTerminationReason] = useState<Exclude<TerminationReason, ''> | ''>('');
  const [terminationNoticeDate, setTerminationNoticeDate] = useState('');
  const [moveOutDate, setMoveOutDate] = useState('');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setTerminationReason('');
    setTerminationNoticeDate('');
    setMoveOutDate('');
    setError(null);
  }, [open]);

  const reasonInvalid = terminationReason === '';
  const dateInvalid = terminationNoticeDate.trim() === '';

  function handleSubmit() {
    if (reasonInvalid || dateInvalid) return;
    setError(null);
    terminateLease.mutate(
      {
        terminationReason: terminationReason as Exclude<TerminationReason, ''>,
        terminationNoticeDate,
        moveOutDate,
        moveOutInspectionAttachmentId: '',
      },
      {
        onSuccess: onSaved,
        onError: (err) => setError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.'),
      },
    );
  }

  return (
    <Dialog fullScreen open={open} onClose={onClose} slots={{ transition: Transition }} aria-labelledby="terminate-lease-title">
      <Stack direction="row" sx={{ alignItems: 'flex-start', justifyContent: 'space-between', px: 3, py: 2 }}>
        <Box>
          <Typography id="terminate-lease-title" variant="h4" sx={{ fontSize: 18 }}>
            Terminate Lease
          </Typography>
          {residentName && <Typography sx={{ fontSize: 12.5, color: tokens.slate[500], mt: 0.25 }}>{residentName}</Typography>}
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
            <Alert severity="warning">
              Ending this lease will automatically mark its unit Vacant. This cannot be undone through this action.
            </Alert>

            <TextField
              select
              label="Termination reason"
              required
              fullWidth
              value={terminationReason}
              onChange={(e) => setTerminationReason(e.target.value as Exclude<TerminationReason, ''>)}
              error={reasonInvalid}
              helperText={reasonInvalid ? 'Required.' : undefined}
            >
              {TERMINATION_REASON_OPTIONS.map((r) => (
                <MenuItem key={r} value={r}>
                  {TERMINATION_REASON_LABELS[r]}
                </MenuItem>
              ))}
            </TextField>
            <TextField
              label="Termination notice date"
              type="date"
              required
              fullWidth
              value={terminationNoticeDate}
              onChange={(e) => setTerminationNoticeDate(e.target.value)}
              error={dateInvalid}
              helperText={dateInvalid ? 'Required.' : undefined}
              slotProps={{ inputLabel: { shrink: true } }}
            />
            <TextField
              label="Move-out date"
              type="date"
              fullWidth
              value={moveOutDate}
              onChange={(e) => setMoveOutDate(e.target.value)}
              placeholder="Optional"
              slotProps={{ inputLabel: { shrink: true } }}
            />
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
            color="error"
            onClick={handleSubmit}
            disabled={terminateLease.isPending || reasonInvalid || dateInvalid}
            startIcon={terminateLease.isPending ? <CircularProgress size={15} color="inherit" /> : undefined}
          >
            {terminateLease.isPending ? 'Terminating…' : 'Terminate lease'}
          </Button>
        </Stack>
      </Stack>
    </Dialog>
  );
}
