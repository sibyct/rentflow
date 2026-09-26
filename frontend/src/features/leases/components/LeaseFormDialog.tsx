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
  LEASE_TYPE_LABELS,
  LEASE_TYPES,
  RENEWAL_STATUS_LABELS,
  RENEWAL_STATUSES,
  TERMINATION_REASON_LABELS,
  TERMINATION_REASON_OPTIONS,
  type LeaseStatus,
} from '../types';
import { ChangeRentDialog } from './ChangeRentDialog';
import { CorrectTerminationDialog } from './CorrectTerminationDialog';
import { GenerateRenewalLeaseDialog } from './GenerateRenewalLeaseDialog';
import { TerminateLeaseDialog } from './TerminateLeaseDialog';

// A lease's Status select never offers 'terminated' — ending a lease
// always goes through the dedicated Terminate Lease action (R10), so
// the unit-vacancy wiring and audit entry always happen with it.
// Beyond that, only forward transitions are offered: a draft lease can
// move to active or stay draft, but an already-active lease can't be
// walked back to draft — there is nothing left to choose, so the field
// renders disabled in that case (see the Status Controller below).
function selectableLeaseStatuses(current: LeaseStatus | undefined): LeaseStatus[] {
  if (current === 'active') return ['active'];
  if (current === 'terminated') return ['terminated'];
  return ['draft', 'active'];
}

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
 * PropertyFormModal/UnitFormDialog. Create and edit render a
 * deliberately different field set from one shared form/schema rather
 * than two components that would drift apart over time:
 *
 * - Status and Move-out date only render in edit mode — a lease that
 *   doesn't exist yet has nothing to move anyone out of, and its
 *   status isn't a create-time user choice (see the Historical entry
 *   section below and onSubmit's create branch).
 * - On create, the only trace of "signed" is the Historical entry
 *   toggle, for importing an already-signed paper lease — checking it
 *   is what makes the new lease Active instead of Draft.
 * - Renewal & termination is gated by the lease's *current status*,
 *   not just edit-vs-create: hidden for a Draft lease (nothing to
 *   renew/terminate yet, same as Create), fully editable for Active
 *   (the only state where Renewal Status/its sub-fields/Terminate
 *   Lease are live), and read-only historical record for a Terminated
 *   lease — the only way to change anything at that point is the
 *   separate, reason-required Correct Termination Details action.
 *   Signed/Signed date sits outside this gating: it's tracked for the
 *   whole edit-mode lifecycle regardless of status.
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
  const [changeRentOpen, setChangeRentOpen] = useState(false);
  const [terminateOpen, setTerminateOpen] = useState(false);
  const [generateRenewalOpen, setGenerateRenewalOpen] = useState(false);
  const [correctTerminationOpen, setCorrectTerminationOpen] = useState(false);

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
  const renewalStatus = watch('renewalStatus');
  const signed = watch('signed');
  const isPending = createLease.isPending || updateLease.isPending;
  const isWaitingForRecord = isEditMode && (isLoadingLease || !existingLease);
  const pickedUnit = units?.find((u) => u.id === effectiveUnitId);
  const title = isEditMode ? 'Edit Lease' : 'Add Lease';
  // Monthly Rent is only directly editable while the lease is still a
  // draft — once it has gone active, a rent change must go through the
  // dedicated Change Rent action instead of a direct field edit (R3).
  const rentLocked = isEditMode && existingLease?.status !== 'draft';

  function onSubmit(values: LeaseFormValues) {
    setSubmitError(null);
    const onError = (err: unknown) =>
      setSubmitError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.');

    if (isEditMode && leaseId && existingLease) {
      updateLease.mutate({ id: leaseId, values, originalStatus: existingLease.status }, { onSuccess: onSaved, onError });
    } else {
      // Status is never a create-time user choice (there's no Status
      // field on this form in create mode) — it's computed here: Draft
      // for the normal new-lease flow, or immediately Active when the
      // Historical entry toggle says this is an already-signed paper
      // lease being recorded after the fact.
      createLease.mutate({ ...values, status: values.signed ? 'active' : 'draft' }, { onSuccess: onSaved, onError });
    }
  }

  return (
    <>
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
                  {/* Move-out date only makes sense once a lease exists
                      to move out of — absent on create. */}
                  {isEditMode && (
                    <Controller
                      name="moveOutDate"
                      control={control}
                      render={({ field }) => <TextField {...field} label="Move-out date" type="date" fullWidth slotProps={{ inputLabel: { shrink: true } }} />}
                    />
                  )}
                </Stack>
                {/* Status isn't a create-time choice at all (see
                    onSubmit) — shown only in edit mode, and even then
                    restricted to the transitions actually valid from
                    the lease's current status: an active lease can't
                    be walked back to draft, and ending a lease always
                    goes through Terminate Lease, never this select. */}
                {isEditMode && (
                  <Controller
                    name="status"
                    control={control}
                    render={({ field }) => {
                      const options = selectableLeaseStatuses(existingLease?.status);
                      return (
                        <TextField
                          {...field}
                          select
                          label="Status"
                          fullWidth
                          disabled={options.length <= 1}
                          helperText={
                            existingLease?.status === 'terminated'
                              ? 'Use "Terminate Lease" to end a lease — this record is already terminated.'
                              : existingLease?.status === 'active'
                                ? 'Use "Terminate Lease" below to end this lease — it can no longer be moved back to draft.'
                                : undefined
                          }
                        >
                          {options.map((s) => (
                            <MenuItem key={s} value={s}>
                              {s === 'active' ? 'Active' : s === 'draft' ? 'Draft' : 'Terminated'}
                            </MenuItem>
                          ))}
                        </TextField>
                      );
                    }}
                  />
                )}
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
                <Stack direction="row" spacing={2}>
                  <Controller
                    name="primaryResidentPhone"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Phone" type="tel" fullWidth placeholder="Optional" />}
                  />
                  <Controller
                    name="primaryResidentEmail"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Email" type="email" fullWidth placeholder="Optional" />}
                  />
                </Stack>
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
                <Stack direction="row" spacing={2} sx={{ alignItems: 'flex-start' }}>
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
                        disabled={rentLocked}
                        slotProps={{ input: { startAdornment: '$' } }}
                        error={Boolean(errors.monthlyRent)}
                        helperText={
                          errors.monthlyRent?.message ?? (rentLocked ? "This is the lease's current effective rent — use Change Rent to update it." : undefined)
                        }
                      />
                    )}
                  />
                  {rentLocked && (
                    <Button
                      variant="outlined"
                      onClick={() => setChangeRentOpen(true)}
                      disabled={existingLease?.status !== 'active'}
                      sx={{ mt: 1, flexShrink: 0, borderColor: tokens.slate[300], color: tokens.slate[700] }}
                    >
                      Change Rent
                    </Button>
                  )}
                  <Controller
                    name="securityDeposit"
                    control={control}
                    render={({ field }) => (
                      <TextField
                        {...field}
                        label="Security deposit"
                        type="number"
                        fullWidth
                        slotProps={{ input: { startAdornment: '$' } }}
                        error={Boolean(errors.securityDeposit)}
                        helperText={errors.securityDeposit?.message}
                      />
                    )}
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
                    render={({ field }) => (
                      <TextField
                        {...field}
                        label="Rent due day"
                        type="number"
                        fullWidth
                        placeholder="1–31"
                        error={Boolean(errors.rentDueDay)}
                        helperText={errors.rentDueDay?.message}
                      />
                    )}
                  />
                </Stack>
                <Stack direction="row" spacing={2}>
                  <Controller
                    name="lateFeeAmount"
                    control={control}
                    render={({ field }) => (
                      <TextField
                        {...field}
                        label="Late fee amount"
                        type="number"
                        fullWidth
                        slotProps={{ input: { startAdornment: '$' } }}
                        error={Boolean(errors.lateFeeAmount)}
                        helperText={errors.lateFeeAmount?.message}
                      />
                    )}
                  />
                  <Controller
                    name="lateFeeGraceDays"
                    control={control}
                    render={({ field }) => (
                      <TextField
                        {...field}
                        label="Late fee grace (days)"
                        type="number"
                        fullWidth
                        error={Boolean(errors.lateFeeGraceDays)}
                        helperText={errors.lateFeeGraceDays?.message}
                      />
                    )}
                  />
                </Stack>
              </FormSection>

              {/* Manual/historical entry only, for importing an
                  already-signed paper lease — checking this is what
                  makes the new lease Active instead of Draft (see
                  onSubmit). This is not part of the normal create
                  flow's Signed tracking, which lives in Renewal &
                  termination and only applies once a lease exists. */}
              {!isEditMode && (
                <FormSection title="Historical entry">
                  <Controller
                    name="signed"
                    control={control}
                    render={({ field }) => (
                      <FormControlLabel
                        control={<Checkbox checked={field.value} onChange={(e) => field.onChange(e.target.checked)} />}
                        label="This lease is already signed (manual entry)"
                      />
                    )}
                  />
                  {signed && (
                    <Controller
                      name="signedDate"
                      control={control}
                      render={({ field }) => <TextField {...field} label="Signed date" type="date" fullWidth slotProps={{ inputLabel: { shrink: true } }} />}
                    />
                  )}
                </FormSection>
              )}

              {/* Signed/Signed date is tracked for the whole edit-mode
                  lifecycle regardless of status — unlike the rest of
                  Renewal & termination below, it isn't gated by Draft/
                  Active/Terminated. */}
              {isEditMode && (
                <FormSection title="Signature">
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

              {/* Renewal & termination is status-gated: a Draft lease
                  hasn't started its lifecycle yet (same as Create, so
                  hidden entirely); an Active lease shows it fully
                  editable, the only state where these fields are live;
                  a Terminated lease shows it as a read-only historical
                  record — Renewal Status, its sub-fields, and
                  termination reason/notice date render as plain text,
                  the Terminate Lease button and the editable notice-
                  date field are both gone, and the only way to change
                  anything is the separate, reason-required Correct
                  termination details action below. */}
              {isEditMode && existingLease?.status === 'active' && (
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

                  {/* "Offered" reveals the proposed terms — stored
                      separately from the active lease's real rent/end
                      date and never applied to it until Accepted +
                      generated (R8/R9). */}
                  {renewalStatus === 'offered' && (
                    <Stack spacing={2} sx={{ p: 2, bgcolor: tokens.slate[50], borderRadius: `${tokens.radiusControl}px` }}>
                      <Stack direction="row" spacing={2}>
                        <Controller
                          name="proposedRent"
                          control={control}
                          render={({ field }) => (
                            <TextField
                              {...field}
                              label="Proposed rent"
                              type="number"
                              required
                              fullWidth
                              slotProps={{ input: { startAdornment: '$' } }}
                              error={Boolean(errors.proposedRent)}
                              helperText={errors.proposedRent?.message}
                            />
                          )}
                        />
                        <Controller
                          name="proposedEndDate"
                          control={control}
                          render={({ field }) => <TextField {...field} label="Proposed end date" type="date" fullWidth slotProps={{ inputLabel: { shrink: true } }} />}
                        />
                      </Stack>
                      <Controller
                        name="offerSentDate"
                        control={control}
                        render={({ field }) => <TextField {...field} label="Offer sent date" type="date" fullWidth slotProps={{ inputLabel: { shrink: true } }} />}
                      />
                    </Stack>
                  )}

                  {/* "Accepted" surfaces Generate Renewal Lease rather
                      than letting the proposed terms silently edit the
                      current lease (R8). */}
                  {renewalStatus === 'accepted' && existingLease && (
                    <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center', p: 2, bgcolor: tokens.slate[50], borderRadius: `${tokens.radiusControl}px` }}>
                      <Typography sx={{ fontSize: 12.5, color: tokens.slate[600], flex: 1 }}>
                        {existingLease.renewedIntoLeaseId
                          ? 'This lease has already been renewed into a new lease record.'
                          : 'Ready to generate the new lease record from the proposed terms.'}
                      </Typography>
                      <Button
                        variant="contained"
                        onClick={() => setGenerateRenewalOpen(true)}
                        disabled={Boolean(existingLease.renewedIntoLeaseId)}
                        sx={{ flexShrink: 0 }}
                      >
                        Generate Renewal Lease
                      </Button>
                    </Stack>
                  )}

                  {/* "Declined" requires a termination reason + notice
                      date, same as an actual termination (R8) — these
                      fields stay visible regardless of renewal status
                      since a lease can also be terminated outright via
                      the dedicated action below. This is also the
                      "Notice Period override" field: the only place a
                      termination notice date is directly editable —
                      once Terminated, it renders read-only instead
                      (see the read-only block below). */}
                  <Stack direction="row" spacing={2}>
                    <Controller
                      name="terminationReason"
                      control={control}
                      render={({ field }) => (
                        <TextField {...field} select label="Termination reason" required={renewalStatus === 'declined'} fullWidth error={Boolean(errors.terminationReason)} helperText={errors.terminationReason?.message}>
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
                      render={({ field }) => (
                        <TextField
                          {...field}
                          label="Termination notice date"
                          type="date"
                          required={renewalStatus === 'declined'}
                          fullWidth
                          error={Boolean(errors.terminationNoticeDate)}
                          helperText={errors.terminationNoticeDate?.message}
                          slotProps={{ inputLabel: { shrink: true } }}
                        />
                      )}
                    />
                  </Stack>

                  <Stack direction="row" sx={{ justifyContent: 'flex-end' }}>
                    <Button variant="outlined" color="error" onClick={() => setTerminateOpen(true)}>
                      Terminate Lease
                    </Button>
                  </Stack>
                </FormSection>
              )}

              {isEditMode && existingLease?.status === 'terminated' && (
                <FormSection title="Renewal & termination">
                  <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>
                    This lease is terminated — the fields below are a read-only historical record.
                  </Typography>
                  <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)' }, gap: 2 }}>
                    <ReadOnlyField label="Renewal status" value={RENEWAL_STATUS_LABELS[existingLease.renewalStatus]} />
                    {existingLease.renewalStatus === 'offered' && (
                      <>
                        <ReadOnlyField label="Proposed rent" value={existingLease.proposedRent != null ? `$${existingLease.proposedRent}` : ''} />
                        <ReadOnlyField label="Proposed end date" value={existingLease.proposedEndDate} />
                        <ReadOnlyField label="Offer sent date" value={existingLease.offerSentDate} />
                      </>
                    )}
                    {existingLease.renewalStatus === 'accepted' && (
                      <ReadOnlyField
                        label="Renewal outcome"
                        value={existingLease.renewedIntoLeaseId ? 'Renewed into a new lease record' : 'Accepted, not generated'}
                      />
                    )}
                    <ReadOnlyField
                      label="Termination reason"
                      value={existingLease.terminationReason ? TERMINATION_REASON_LABELS[existingLease.terminationReason] : ''}
                    />
                    <ReadOnlyField label="Termination notice date" value={existingLease.terminationNoticeDate} />
                  </Box>

                  <Stack direction="row" sx={{ justifyContent: 'flex-end' }}>
                    <Button variant="outlined" onClick={() => setCorrectTerminationOpen(true)} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
                      Correct Termination Details
                    </Button>
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

    {isEditMode && leaseId && existingLease && (
      <>
        <ChangeRentDialog
          open={changeRentOpen}
          onClose={() => setChangeRentOpen(false)}
          leaseId={leaseId}
          unitId={existingLease.unitId}
          currentAmount={existingLease.monthlyRent}
          onSaved={() => setChangeRentOpen(false)}
        />
        <TerminateLeaseDialog
          open={terminateOpen}
          onClose={() => setTerminateOpen(false)}
          leaseId={leaseId}
          unitId={existingLease.unitId}
          residentName={existingLease.primaryResidentName}
          onSaved={() => setTerminateOpen(false)}
        />
        <GenerateRenewalLeaseDialog
          open={generateRenewalOpen}
          onClose={() => setGenerateRenewalOpen(false)}
          sourceLease={existingLease}
          onSaved={() => {
            setGenerateRenewalOpen(false);
            onClose();
            onSaved();
          }}
        />
        <CorrectTerminationDialog
          open={correctTerminationOpen}
          onClose={() => setCorrectTerminationOpen(false)}
          lease={existingLease}
          onSaved={() => setCorrectTerminationOpen(false)}
        />
      </>
    )}
    </>
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

/** A plain-text row for a Terminated lease's read-only Renewal & termination history — never an editable control, see the correction action for how these values actually change. */
function ReadOnlyField({ label, value }: { label: string; value: string }) {
  return (
    <Box>
      <Typography sx={{ fontSize: 11, fontWeight: 600, letterSpacing: '0.04em', color: tokens.slate[500], textTransform: 'uppercase' }}>
        {label}
      </Typography>
      <Typography sx={{ fontSize: 13.5, color: value ? tokens.slate[700] : tokens.slate[400], mt: 0.25 }}>{value || '—'}</Typography>
    </Box>
  );
}
