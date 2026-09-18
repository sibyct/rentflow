import { forwardRef, useEffect, useState, type ReactNode } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogContent from '@mui/material/DialogContent';
import Divider from '@mui/material/Divider';
import FormControlLabel from '@mui/material/FormControlLabel';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Slide from '@mui/material/Slide';
import type { TransitionProps } from '@mui/material/transitions';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import CloseOutlined from '@mui/icons-material/CloseOutlined';
import ExpandMoreOutlined from '@mui/icons-material/ExpandMoreOutlined';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import type { UnitRowWithProperty } from '@/features/units/types';
import { toFormValues } from '../api/leasesApi';
import { useCreateLease, useLease, useUpdateLease } from '../hooks/useLeasesQueries';
import { leaseDefaultValues, leaseSchema, type LeaseFormValues } from '../schemas/leaseSchema';
import {
  DEPOSIT_STATUS_LABELS,
  DEPOSIT_STATUS_OPTIONS,
  LEASE_STATUSES,
  LEASE_TYPE_LABELS,
  LEASE_TYPES,
  RENEWAL_STATUS_LABELS,
  RENEWAL_STATUSES,
  TERMINATION_REASON_LABELS,
  TERMINATION_REASON_OPTIONS,
} from '../types';

const Transition = forwardRef(function Transition(
  props: TransitionProps & { children: React.ReactElement },
  ref: React.Ref<unknown>,
) {
  return <Slide direction="up" ref={ref} {...props} />;
});

interface LeaseFormDialogProps {
  open: boolean;
  onClose: () => void;
  /** Fixed unit context (e.g. opened from a unit's "Create Lease" button) — no picker shown. Omit only for the global Leases page's Add Lease, and pass `units` so there's something to pick from. */
  unitId?: string;
  /** Required when unitId is omitted. */
  units?: UnitRowWithProperty[];
  /** Present = edit an existing lease; absent = create a new one. */
  leaseId?: string;
  onSaved: () => void;
}

/**
 * Single-lease add/edit popover — same Dialog chrome as
 * PropertyFormModal/UnitFormDialog. Renewal & termination fields only
 * appear once editing an existing lease: you don't terminate a lease
 * you're still creating, so showing them on the create form would just
 * be dead UI.
 */
export function LeaseFormDialog({ open, onClose, unitId, units, leaseId, onSaved }: LeaseFormDialogProps) {
  const isEditMode = Boolean(leaseId);
  const [pickedUnitId, setPickedUnitId] = useState('');
  const effectiveUnitId = unitId ?? pickedUnitId;
  const needsUnitPicker = !unitId && !pickedUnitId;

  const { data: existingLease, isLoading: isLoadingLease } = useLease(leaseId);
  const createLease = useCreateLease(effectiveUnitId);
  const updateLease = useUpdateLease(effectiveUnitId);
  const [notesOpen, setNotesOpen] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const {
    control,
    handleSubmit,
    reset,
    watch,
    formState: { errors },
  } = useForm<LeaseFormValues>({
    resolver: zodResolver(leaseSchema),
    defaultValues: leaseDefaultValues(),
  });

  useEffect(() => {
    if (!open) return;
    if (isEditMode) {
      if (existingLease) reset(toFormValues(existingLease));
    } else {
      reset(leaseDefaultValues());
    }
    setPickedUnitId('');
    setSubmitError(null);
    setNotesOpen(false);
  }, [open, isEditMode, existingLease, reset]);

  const leaseType = watch('leaseType');
  const isPending = createLease.isPending || updateLease.isPending;
  const isWaitingForRecord = isEditMode && (isLoadingLease || !existingLease);
  const pickedUnit = units?.find((u) => u.id === effectiveUnitId);
  const title = isEditMode ? 'Edit Lease' : 'Add Lease';

  function onSubmit(values: LeaseFormValues) {
    setSubmitError(null);
    const onError = (err: unknown) =>
      setSubmitError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.');

    if (isEditMode && leaseId) {
      updateLease.mutate({ id: leaseId, values }, { onSuccess: onSaved, onError });
    } else {
      createLease.mutate(values, { onSuccess: onSaved, onError });
    }
  }

  return (
    <Dialog fullScreen open={open} onClose={onClose} slots={{ transition: Transition }} aria-labelledby="lease-form-title">
      <Stack direction="row" sx={{ alignItems: 'flex-start', justifyContent: 'space-between', px: 3, py: 2 }}>
        <Box>
          <Typography id="lease-form-title" variant="h4" sx={{ fontSize: 18 }}>
            {title}
          </Typography>
          {(pickedUnit || existingLease) && (
            <Typography sx={{ fontSize: 12.5, color: tokens.slate[500], mt: 0.25 }}>
              {existingLease ? `${existingLease.propertyName} / ${existingLease.unitName}` : `${pickedUnit!.propertyName} / ${pickedUnit!.unitName}`}
            </Typography>
          )}
        </Box>
        <IconButton onClick={onClose} aria-label="Close">
          <CloseOutlined />
        </IconButton>
      </Stack>
      <Divider />

      <DialogContent sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', overflowX: 'hidden' }}>
        {isWaitingForRecord ? (
          <Box sx={{ py: 8 }}>
            <Typography sx={{ fontSize: 13.5, color: tokens.slate[500] }}>Loading lease…</Typography>
          </Box>
        ) : needsUnitPicker ? (
          <Stack spacing={2} sx={{ width: '100%', maxWidth: 440, py: 8 }}>
            <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>Which unit is this lease for?</Typography>
            <TextField select label="Unit" value={pickedUnitId} onChange={(e) => setPickedUnitId(e.target.value)} fullWidth autoFocus>
              {(units ?? []).map((u) => (
                <MenuItem key={u.id} value={u.id}>
                  {u.propertyName} — {u.unitName}
                </MenuItem>
              ))}
            </TextField>
          </Stack>
        ) : (
          <Box component="form" id="lease-form" onSubmit={handleSubmit(onSubmit)} sx={{ width: '100%', maxWidth: 640, py: 3 }}>
            <Stack spacing={3}>
              {submitError && <Alert severity="error">{submitError}</Alert>}

              <FormSection title="Terms">
                <Controller
                  name="leaseType"
                  control={control}
                  render={({ field }) => (
                    <TextField {...field} value={field.value ?? ''} select label="Lease type" required fullWidth error={Boolean(errors.leaseType)} helperText={errors.leaseType?.message}>
                      {LEASE_TYPES.map((t) => (
                        <MenuItem key={t} value={t}>
                          {LEASE_TYPE_LABELS[t]}
                        </MenuItem>
                      ))}
                    </TextField>
                  )}
                />
                <Stack direction="row" spacing={2}>
                  <Controller
                    name="startDate"
                    control={control}
                    render={({ field }) => (
                      <TextField
                        {...field}
                        label="Start date"
                        type="date"
                        required
                        fullWidth
                        slotProps={{ inputLabel: { shrink: true } }}
                        error={Boolean(errors.startDate)}
                        helperText={errors.startDate?.message}
                      />
                    )}
                  />
                  <Controller
                    name="endDate"
                    control={control}
                    render={({ field }) => (
                      <TextField
                        {...field}
                        label="End date"
                        type="date"
                        required={leaseType === 'fixed'}
                        fullWidth
                        slotProps={{ inputLabel: { shrink: true } }}
                        error={Boolean(errors.endDate)}
                        helperText={errors.endDate?.message ?? (leaseType === 'month_to_month' ? 'Runs until terminated' : undefined)}
                      />
                    )}
                  />
                </Stack>
                <Stack direction="row" spacing={2}>
                  <Controller
                    name="moveInDate"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Move-in date" type="date" fullWidth slotProps={{ inputLabel: { shrink: true } }} />}
                  />
                  <Controller
                    name="moveOutDate"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Move-out date" type="date" fullWidth slotProps={{ inputLabel: { shrink: true } }} />}
                  />
                </Stack>
                <Controller
                  name="status"
                  control={control}
                  render={({ field }) => (
                    <TextField {...field} select label="Status" fullWidth>
                      {LEASE_STATUSES.map((s) => (
                        <MenuItem key={s} value={s}>
                          {s === 'active' ? 'Active' : s === 'draft' ? 'Draft' : 'Terminated'}
                        </MenuItem>
                      ))}
                    </TextField>
                  )}
                />
              </FormSection>

              <FormSection title="Parties">
                <Controller
                  name="primaryResidentName"
                  control={control}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Primary resident"
                      required
                      fullWidth
                      error={Boolean(errors.primaryResidentName)}
                      helperText={errors.primaryResidentName?.message}
                    />
                  )}
                />
                <Controller
                  name="coResidents"
                  control={control}
                  render={({ field }) => <TextField {...field} label="Co-residents / co-signers" fullWidth placeholder="Comma-separated names (optional)" />}
                />
                <Controller
                  name="emergencyContact"
                  control={control}
                  render={({ field }) => <TextField {...field} label="Emergency contact" fullWidth placeholder="Optional" />}
                />
              </FormSection>

              <FormSection title="Financials">
                <Stack direction="row" spacing={2}>
                  <Controller
                    name="monthlyRent"
                    control={control}
                    render={({ field }) => (
                      <TextField
                        {...field}
                        label="Monthly rent"
                        type="number"
                        required
                        fullWidth
                        slotProps={{ input: { startAdornment: '$' } }}
                        error={Boolean(errors.monthlyRent)}
                        helperText={errors.monthlyRent?.message}
                      />
                    )}
                  />
                  <Controller
                    name="securityDeposit"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Security deposit" type="number" fullWidth slotProps={{ input: { startAdornment: '$' } }} />}
                  />
                </Stack>
                <Stack direction="row" spacing={2}>
                  <Controller
                    name="depositStatus"
                    control={control}
                    render={({ field }) => (
                      <TextField {...field} select label="Deposit status" fullWidth>
                        <MenuItem value="">
                          <em>Not specified</em>
                        </MenuItem>
                        {DEPOSIT_STATUS_OPTIONS.map((s) => (
                          <MenuItem key={s} value={s}>
                            {DEPOSIT_STATUS_LABELS[s]}
                          </MenuItem>
                        ))}
                      </TextField>
                    )}
                  />
                  <Controller
                    name="rentDueDay"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Rent due day" type="number" fullWidth placeholder="1–31" />}
                  />
                </Stack>
                <Stack direction="row" spacing={2}>
                  <Controller
                    name="lateFeeAmount"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Late fee amount" type="number" fullWidth slotProps={{ input: { startAdornment: '$' } }} />}
                  />
                  <Controller
                    name="lateFeeGraceDays"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Late fee grace (days)" type="number" fullWidth />}
                  />
                </Stack>
              </FormSection>

              {isEditMode && (
                <FormSection title="Renewal & termination">
                  <Controller
                    name="renewalStatus"
                    control={control}
                    render={({ field }) => (
                      <TextField {...field} select label="Renewal status" fullWidth>
                        {RENEWAL_STATUSES.map((s) => (
                          <MenuItem key={s} value={s}>
                            {RENEWAL_STATUS_LABELS[s]}
                          </MenuItem>
                        ))}
                      </TextField>
                    )}
                  />
                  <Stack direction="row" spacing={2}>
                    <Controller
                      name="terminationReason"
                      control={control}
                      render={({ field }) => (
                        <TextField {...field} select label="Termination reason" fullWidth>
                          <MenuItem value="">
                            <em>Not applicable</em>
                          </MenuItem>
                          {TERMINATION_REASON_OPTIONS.map((r) => (
                            <MenuItem key={r} value={r}>
                              {TERMINATION_REASON_LABELS[r]}
                            </MenuItem>
                          ))}
                        </TextField>
                      )}
                    />
                    <Controller
                      name="terminationNoticeDate"
                      control={control}
                      render={({ field }) => <TextField {...field} label="Termination notice date" type="date" fullWidth slotProps={{ inputLabel: { shrink: true } }} />}
                    />
                  </Stack>
                  <Stack direction="row" spacing={2} sx={{ alignItems: 'center' }}>
                    <Controller
                      name="signed"
                      control={control}
                      render={({ field }) => (
                        <FormControlLabel
                          control={<Checkbox checked={field.value} onChange={(e) => field.onChange(e.target.checked)} />}
                          label="Signed"
                        />
                      )}
                    />
                    <Controller
                      name="signedDate"
                      control={control}
                      render={({ field }) => <TextField {...field} label="Signed date" type="date" fullWidth slotProps={{ inputLabel: { shrink: true } }} />}
                    />
                  </Stack>
                </FormSection>
              )}

              <Box>
                <Link
                  component="button"
                  type="button"
                  onClick={() => setNotesOpen((v) => !v)}
                  sx={{ display: 'flex', alignItems: 'center', gap: 0.5, fontSize: 12.5, fontWeight: 600 }}
                >
                  Notes
                  <ExpandMoreOutlined sx={{ fontSize: 18, transform: notesOpen ? 'rotate(180deg)' : 'none', transition: 'transform 0.15s' }} />
                </Link>
                {notesOpen && (
                  <Box sx={{ mt: 1.5 }}>
                    <Controller
                      name="notes"
                      control={control}
                      render={({ field }) => <TextField {...field} label="Notes" fullWidth multiline minRows={3} />}
                    />
                  </Box>
                )}
              </Box>
            </Stack>
          </Box>
        )}
      </DialogContent>

      {!needsUnitPicker && !isWaitingForRecord && (
        <>
          <Divider />
          <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'flex-end', px: 3, py: 2 }}>
            <Stack direction="row" spacing={1.5}>
              <Button variant="outlined" onClick={onClose} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
                Cancel
              </Button>
              <Button
                type="submit"
                form="lease-form"
                variant="contained"
                disabled={isPending}
                startIcon={isPending ? <CircularProgress size={15} color="inherit" /> : undefined}
              >
                {isPending ? 'Saving…' : isEditMode ? 'Save changes' : 'Add Lease'}
              </Button>
            </Stack>
          </Stack>
        </>
      )}
    </Dialog>
  );
}

function FormSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Stack spacing={1.5}>
      <Typography sx={{ fontSize: 11, fontWeight: 600, letterSpacing: '0.04em', color: tokens.slate[500], textTransform: 'uppercase' }}>
        {title}
      </Typography>
      {children}
    </Stack>
  );
}
