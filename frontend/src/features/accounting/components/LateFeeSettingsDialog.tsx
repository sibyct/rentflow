import { useEffect, useState } from 'react';
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
import { centsToDollarsInput, dollarsToCents } from '@/shared/lib/format';
import { useAccountingSettings, useUpdateAccountingSettings } from '../hooks/useAccountingQueries';
import type { AccountingSettings } from '../types';

interface LateFeeSettingsDialogProps {
  open: boolean;
  onClose: () => void;
  onSaved: () => void;
}

interface Draft {
  kind: AccountingSettings['lateFeeKind'];
  /** Dollars when flat, percent when percent. */
  value: string;
  graceDays: string;
  dueDay: string;
}

function toDraft(s: AccountingSettings): Draft {
  return {
    kind: s.lateFeeKind,
    value: s.lateFeeKind === 'flat' ? centsToDollarsInput(s.lateFeeValue) : String(s.lateFeeValue / 100),
    graceDays: String(s.graceDays),
    dueDay: String(s.defaultRentDueDay),
  };
}

function toSettings(d: Draft): { settings?: AccountingSettings; errors: Partial<Record<keyof Draft, string>> } {
  const errors: Partial<Record<keyof Draft, string>> = {};
  const raw = Number(d.value.trim() === '' ? '0' : d.value);
  if (!Number.isFinite(raw) || raw < 0) errors.value = 'Enter zero or a positive number.';
  else if (d.kind === 'percent' && raw > 100) errors.value = 'A percentage cannot exceed 100.';

  const grace = Number(d.graceDays);
  if (!Number.isInteger(grace) || grace < 0 || grace > 60) errors.graceDays = 'Whole days, 0–60.';
  const due = Number(d.dueDay);
  if (!Number.isInteger(due) || due < 1 || due > 31) errors.dueDay = 'A day of the month, 1–31.';
  if (Object.keys(errors).length > 0) return { errors };

  const value = d.kind === 'flat' ? (dollarsToCents(d.value || '0') ?? 0) : Math.round(raw * 100);
  return { errors, settings: { lateFeeKind: d.kind, lateFeeValue: value, graceDays: grace, defaultRentDueDay: due } };
}

/** The account-wide late-fee rule: flat or percent, after N grace days. A lease's own late fee / grace / due day overrides it. */
export function LateFeeSettingsDialog({ open, onClose, onSaved }: LateFeeSettingsDialogProps) {
  const { data: settings } = useAccountingSettings();
  const update = useUpdateAccountingSettings();
  const [draft, setDraft] = useState<Draft | null>(null);
  const [errors, setErrors] = useState<Partial<Record<keyof Draft, string>>>({});
  const [submitError, setSubmitError] = useState<string | null>(null);

  useEffect(() => {
    if (open && settings) {
      setDraft(toDraft(settings));
      setErrors({});
      setSubmitError(null);
    }
  }, [open, settings]);

  function submit() {
    if (!draft) return;
    const result = toSettings(draft);
    setErrors(result.errors);
    if (!result.settings) return;
    update.mutate(result.settings, {
      onSuccess: onSaved,
      onError: (err) => setSubmitError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.'),
    });
  }

  return (
    <Dialog open={open} onClose={update.isPending ? undefined : onClose} fullWidth maxWidth="xs">
      <DialogTitle>Late fee rules</DialogTitle>
      <DialogContent>
        {draft && (
          <Stack spacing={2} sx={{ pt: 0.5 }}>
            <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>
              When rent is still unpaid after the grace period, a late-fee line is added to that month&apos;s balance
              automatically. A lease&apos;s own late fee, grace days or due day takes priority over these defaults. Set the
              amount to 0 to turn late fees off.
            </Typography>
            {submitError && <Alert severity="error">{submitError}</Alert>}
            <Stack direction="row" spacing={2}>
              <TextField
                select
                label="Fee type"
                value={draft.kind}
                onChange={(e) => setDraft({ ...draft, kind: e.target.value as Draft['kind'] })}
                sx={{ flex: 1 }}
              >
                <MenuItem value="flat">Flat amount</MenuItem>
                <MenuItem value="percent">% of monthly rent</MenuItem>
              </TextField>
              <TextField
                label={draft.kind === 'flat' ? 'Amount' : 'Percent'}
                type="number"
                value={draft.value}
                onChange={(e) => setDraft({ ...draft, value: e.target.value })}
                error={Boolean(errors.value)}
                helperText={errors.value}
                sx={{ flex: 1 }}
                slotProps={{
                  input: draft.kind === 'flat' ? { startAdornment: '$' } : { endAdornment: '%' },
                  htmlInput: { step: draft.kind === 'flat' ? '0.01' : '0.1', min: 0 },
                }}
              />
            </Stack>
            <Stack direction="row" spacing={2}>
              <TextField
                label="Grace days"
                type="number"
                value={draft.graceDays}
                onChange={(e) => setDraft({ ...draft, graceDays: e.target.value })}
                error={Boolean(errors.graceDays)}
                helperText={errors.graceDays ?? 'Days after the due date'}
                sx={{ flex: 1 }}
              />
              <TextField
                label="Default rent due day"
                type="number"
                value={draft.dueDay}
                onChange={(e) => setDraft({ ...draft, dueDay: e.target.value })}
                error={Boolean(errors.dueDay)}
                helperText={errors.dueDay ?? 'When a lease sets none'}
                sx={{ flex: 1 }}
              />
            </Stack>
          </Stack>
        )}
      </DialogContent>
      <DialogActions sx={{ p: 2.5, pt: 0 }}>
        <Button variant="text" onClick={onClose} disabled={update.isPending} sx={{ color: tokens.slate[600] }}>
          Cancel
        </Button>
        <Button
          variant="contained"
          onClick={submit}
          disabled={update.isPending || !draft}
          startIcon={update.isPending ? <CircularProgress size={15} color="inherit" /> : undefined}
        >
          {update.isPending ? 'Saving…' : 'Save rules'}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
