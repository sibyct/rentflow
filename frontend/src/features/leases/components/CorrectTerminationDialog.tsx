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
import { useCorrectTermination } from '../hooks/useLeasesQueries';
import { TERMINATION_REASON_LABELS, TERMINATION_REASON_OPTIONS, type LeaseDetailWithUnitProperty, type TerminationReason } from '../types';

const Transition = forwardRef(function Transition(
  props: TransitionProps & { children: React.ReactElement },
  ref: React.Ref<unknown>,
) {
  return <Slide direction="up" ref={ref} {...props} />;
});

interface CorrectTerminationDialogProps {
  open: boolean;
  onClose: () => void;
  lease: LeaseDetailWithUnitProperty;
  onSaved: () => void;
}

/**
 * A separate, explicitly-labeled action for fixing a data-entry
 * mistake in an already-terminated lease's termination record —
 * never a silent inline edit. Pre-filled from the lease's current
 * termination details so a correction is a small delta, not a blank
 * re-entry, but a reason explaining the correction is mandatory and
 * every use is captured in the lease's audit trail (R18).
 */
export function CorrectTerminationDialog({ open, onClose, lease, onSaved }: CorrectTerminationDialogProps) {
  const correctTermination = useCorrectTermination(lease.id, lease.unitId);
  const [terminationReason, setTerminationReason] = useState<Exclude<TerminationReason, ''> | ''>('');
  const [terminationNoticeDate, setTerminationNoticeDate] = useState('');
  const [moveOutDate, setMoveOutDate] = useState('');
  const [reason, setReason] = useState('');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setTerminationReason(lease.terminationReason);
    setTerminationNoticeDate(lease.terminationNoticeDate);
    setMoveOutDate(lease.moveOutDate);
    setReason('');
    setError(null);
  }, [open, lease]);

  const reasonInvalid = terminationReason === '';
  const dateInvalid = terminationNoticeDate.trim() === '';
  const correctionReasonInvalid = reason.trim() === '';

  function handleSubmit() {
    if (reasonInvalid || dateInvalid || correctionReasonInvalid) return;
    setError(null);
    correctTermination.mutate(
      {
        terminationReason: terminationReason as Exclude<TerminationReason, ''>,
        terminationNoticeDate,
        moveOutDate,
        reason,
      },
      {
        onSuccess: onSaved,
        onError: (err) => setError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.'),
      },
    );
  }

  return (
    <Dialog fullScreen open={open} onClose={onClose} slots={{ transition: Transition }} aria-labelledby="correct-termination-title">
      <Stack direction="row" sx={{ alignItems: 'flex-start', justifyContent: 'space-between', px: 3, py: 2 }}>
        <Box>
          <Typography id="correct-termination-title" variant="h4" sx={{ fontSize: 18 }}>
            Correct Termination Details
          </Typography>
          <Typography sx={{ fontSize: 12.5, color: tokens.slate[500], mt: 0.25 }}>{lease.primaryResidentName}</Typography>
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
              This lease is already terminated. Only use this to fix a data-entry mistake in its termination record —
              not to reverse or re-do the termination itself.
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
            <TextField
              label="Reason for this correction"
              required
              fullWidth
              multiline
              minRows={2}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="e.g. original entry used the wrong reason code"
              error={correctionReasonInvalid}
              helperText={correctionReasonInvalid ? 'Required — explain why these details are being corrected.' : undefined}
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
            onClick={handleSubmit}
            disabled={correctTermination.isPending || reasonInvalid || dateInvalid || correctionReasonInvalid}
            startIcon={correctTermination.isPending ? <CircularProgress size={15} color="inherit" /> : undefined}
          >
            {correctTermination.isPending ? 'Saving…' : 'Save correction'}
          </Button>
        </Stack>
      </Stack>
    </Dialog>
  );
}
