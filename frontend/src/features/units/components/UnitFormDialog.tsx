import { forwardRef, useEffect, useState, type ReactNode } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogContent from '@mui/material/DialogContent';
import Divider from '@mui/material/Divider';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Slide from '@mui/material/Slide';
import type { TransitionProps } from '@mui/material/transitions';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import CloseOutlined from '@mui/icons-material/CloseOutlined';
import ExpandMoreOutlined from '@mui/icons-material/ExpandMoreOutlined';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import { LeaseFormDialog } from '@/features/leases/components/LeaseFormDialog';
import { useLeasesPortfolio } from '@/features/leases/hooks/useLeasesQueries';
import { LEASE_DISPLAY_STATUS_LABELS } from '@/features/leases/types';
import type { PropertyRow } from '@/features/properties/mock/propertyRows';
import { toFormValues } from '../api/unitsApi';
import { useCreateUnit, useUnit, useUpdateUnit } from '../hooks/useUnitsQueries';
import { unitDefaultValues, unitSchema, type UnitFormValues } from '../schemas/unitSchema';
import { UNIT_FURNISHED_LABELS, UNIT_FURNISHED_OPTIONS, UNIT_STATUS_LABELS, UNIT_STATUSES, UNIT_TYPE_LABELS, UNIT_TYPES } from '../types';

const Transition = forwardRef(function Transition(
  props: TransitionProps & { children: React.ReactElement },
  ref: React.Ref<unknown>,
) {
  return <Slide direction="up" ref={ref} {...props} />;
});

interface UnitFormDialogProps {
  open: boolean;
  onClose: () => void;
  /** Fixed property context (property-scoped Add/Edit) — no picker shown. Omit only for the global Units page's Add Unit, and pass `properties` so there's something to pick from. */
  propertyId?: string;
  /** Required when propertyId is omitted. */
  properties?: PropertyRow[];
  /** Present = edit an existing unit; absent = create a new one. */
  unitId?: string;
  onSaved: () => void;
}

/**
 * Single-unit add/edit popover — the same Dialog chrome as
 * PropertyFormModal (full-screen, slide-up, header + close button,
 * footer actions) so the two features read as one design language, but
 * deliberately not a stepper: units have far fewer fields than a
 * property and are often added one after another. See
 * BulkAddUnitsDialog for the spreadsheet-style flow that covers "add a
 * dozen units at once" instead.
 *
 * Opened with no propertyId (only the global Units page's Add Unit does
 * this — everywhere else has a property already in context), the
 * dialog opens showing only a Property picker; every other field
 * appears in the same popover only once one is chosen.
 */
export function UnitFormDialog({ open, onClose, propertyId, properties, unitId, onSaved }: UnitFormDialogProps) {
  const isEditMode = Boolean(unitId);
  const [pickedPropertyId, setPickedPropertyId] = useState('');
  const effectivePropertyId = propertyId ?? pickedPropertyId;
  const needsPropertyPicker = !propertyId && !pickedPropertyId;

  const { data: existingUnit, isLoading: isLoadingUnit } = useUnit(unitId);
  const createUnit = useCreateUnit(effectivePropertyId);
  const updateUnit = useUpdateUnit(effectivePropertyId);
  const [notesOpen, setNotesOpen] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [leaseFormOpen, setLeaseFormOpen] = useState(false);

  // Whichever lease is currently active for this unit, if any — mirrors
  // UnitDetailScreen's own lookup. Without this, an occupied unit that
  // already has a lease looked like it had none: this form only ever
  // offered "Create Lease", with no sign the active one existed.
  const { data: leaseData } = useLeasesPortfolio(
    { unitId, status: 'active', limit: 1, offset: 0 },
    { enabled: isEditMode && Boolean(unitId) },
  );
  const activeLease = leaseData?.leases[0];

  const {
    control,
    handleSubmit,
    reset,
    watch,
    formState: { errors },
  } = useForm<UnitFormValues>({
    resolver: zodResolver(unitSchema),
    defaultValues: unitDefaultValues(),
  });

  // Re-seed every time the dialog opens: a fresh blank form (and, in
  // global mode, a fresh unpicked property) for create, or the fetched
  // unit's values for edit — so reopening for a second unit never
  // carries over the previous one.
  useEffect(() => {
    if (!open) return;
    if (isEditMode) {
      if (existingUnit) reset(toFormValues(existingUnit));
    } else {
      reset(unitDefaultValues());
    }
    setPickedPropertyId('');
    setSubmitError(null);
    setNotesOpen(false);
  }, [open, isEditMode, existingUnit, reset]);

  const status = watch('status');
  const isPending = createUnit.isPending || updateUnit.isPending;
  const isWaitingForRecord = isEditMode && (isLoadingUnit || !existingUnit);
  const pickedProperty = properties?.find((p) => p.id === effectivePropertyId);
  const title = isEditMode ? 'Edit Unit' : 'Add Unit';

  function onSubmit(values: UnitFormValues) {
    setSubmitError(null);
    const onError = (err: unknown) =>
      setSubmitError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.');

    if (isEditMode && unitId) {
      updateUnit.mutate({ id: unitId, values }, { onSuccess: onSaved, onError });
    } else {
      createUnit.mutate(values, { onSuccess: onSaved, onError });
    }
  }

  return (
    <>
      <Dialog fullScreen open={open} onClose={onClose} slots={{ transition: Transition }} aria-labelledby="unit-form-title">
      <Stack direction="row" sx={{ alignItems: 'flex-start', justifyContent: 'space-between', px: 3, py: 2 }}>
        <Box>
          <Typography id="unit-form-title" variant="h4" sx={{ fontSize: 18 }}>
            {title}
          </Typography>
          {pickedProperty && (
            <Typography sx={{ fontSize: 12.5, color: tokens.slate[500], mt: 0.25 }}>{pickedProperty.name}</Typography>
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
            <Typography sx={{ fontSize: 13.5, color: tokens.slate[500] }}>Loading unit…</Typography>
          </Box>
        ) : needsPropertyPicker ? (
          <Stack spacing={2} sx={{ width: '100%', maxWidth: 440, py: 8 }}>
            <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>Which property is this unit part of?</Typography>
            <TextField select label="Property" value={pickedPropertyId} onChange={(e) => setPickedPropertyId(e.target.value)} fullWidth autoFocus>
              {(properties ?? []).map((p) => (
                <MenuItem key={p.id} value={p.id}>
                  {p.name}
                </MenuItem>
              ))}
            </TextField>
          </Stack>
        ) : (
          <Box component="form" id="unit-form" onSubmit={handleSubmit(onSubmit)} sx={{ width: '100%', maxWidth: 640, py: 3 }}>
            <Stack spacing={3}>
              {submitError && <Alert severity="error">{submitError}</Alert>}

              <FormSection title="Identity">
                <Controller
                  name="unitName"
                  control={control}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Unit name"
                      required
                      fullWidth
                      placeholder="e.g. 1A, Suite 200"
                      error={Boolean(errors.unitName)}
                      helperText={errors.unitName?.message}
                    />
                  )}
                />
                <Controller
                  name="floor"
                  control={control}
                  render={({ field }) => <TextField {...field} label="Floor" fullWidth placeholder="Optional" />}
                />
              </FormSection>

              <FormSection title="Specs">
                <Controller
                  name="type"
                  control={control}
                  render={({ field }) => (
                    <TextField {...field} value={field.value ?? ''} select label="Unit type" required fullWidth error={Boolean(errors.type)} helperText={errors.type?.message}>
                      {UNIT_TYPES.map((t) => (
                        <MenuItem key={t} value={t}>
                          {UNIT_TYPE_LABELS[t]}
                        </MenuItem>
                      ))}
                    </TextField>
                  )}
                />
                <Stack direction="row" spacing={2}>
                  <Controller
                    name="bedrooms"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Bedrooms" type="number" sx={{ flex: 1 }} />}
                  />
                  <Controller
                    name="bathrooms"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Bathrooms" type="number" sx={{ flex: 1 }} />}
                  />
                  <Controller
                    name="sqft"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Sq ft" type="number" sx={{ flex: 1 }} />}
                  />
                </Stack>
                <Controller
                  name="furnished"
                  control={control}
                  render={({ field }) => (
                    <TextField {...field} select label="Furnished" fullWidth>
                      <MenuItem value="">
                        <em>Not specified</em>
                      </MenuItem>
                      {UNIT_FURNISHED_OPTIONS.map((f) => (
                        <MenuItem key={f} value={f}>
                          {UNIT_FURNISHED_LABELS[f]}
                        </MenuItem>
                      ))}
                    </TextField>
                  )}
                />
              </FormSection>

              <FormSection title="Status">
                <Controller
                  name="status"
                  control={control}
                  render={({ field }) => (
                    <TextField {...field} select label="Status" fullWidth>
                      {UNIT_STATUSES.map((s) => (
                        <MenuItem key={s} value={s}>
                          {UNIT_STATUS_LABELS[s]}
                        </MenuItem>
                      ))}
                    </TextField>
                  )}
                />
                {status === 'occupied' && (
                  <Stack spacing={1.5} sx={{ p: 2, bgcolor: tokens.slate[50], borderRadius: `${tokens.radiusControl}px` }}>
                    <Controller
                      name="tenantName"
                      control={control}
                      render={({ field }) => (
                        <TextField {...field} label="Tenant" fullWidth placeholder="Who's occupying this unit?" size="small" />
                      )}
                    />
                    {isEditMode && activeLease ? (
                      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', gap: 1.5 }}>
                        <Box sx={{ minWidth: 0 }}>
                          <Typography sx={{ fontSize: 12.5, fontWeight: 600, color: tokens.slate[700] }}>{activeLease.primaryResidentName}</Typography>
                          <Typography sx={{ fontSize: 11.5, color: tokens.slate[500] }}>{LEASE_DISPLAY_STATUS_LABELS[activeLease.displayStatus]} lease</Typography>
                        </Box>
                        <Button
                          size="small"
                          variant="outlined"
                          onClick={() => setLeaseFormOpen(true)}
                          sx={{ borderColor: tokens.slate[300], color: tokens.slate[700], flexShrink: 0 }}
                        >
                          Manage Lease
                        </Button>
                      </Stack>
                    ) : isEditMode ? (
                      <Button
                        size="small"
                        variant="outlined"
                        onClick={() => setLeaseFormOpen(true)}
                        sx={{ borderColor: tokens.slate[300], color: tokens.slate[700], alignSelf: 'flex-start' }}
                      >
                        Create Lease
                      </Button>
                    ) : (
                      <Tooltip title="Save this unit first — a lease needs a real unit to attach to.">
                        <span>
                          <Button size="small" variant="outlined" disabled sx={{ borderColor: tokens.slate[300], color: tokens.slate[500] }}>
                            Create Lease
                          </Button>
                        </span>
                      </Tooltip>
                    )}
                  </Stack>
                )}
              </FormSection>

              <FormSection title="Rent">
                <Stack direction="row" spacing={2}>
                  <Controller
                    name="marketRent"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Market rent" type="number" sx={{ flex: 1 }} slotProps={{ input: { startAdornment: '$' } }} />}
                  />
                  <Controller
                    name="currentRent"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Current rent" type="number" sx={{ flex: 1 }} slotProps={{ input: { startAdornment: '$' } }} />}
                  />
                </Stack>
                <Stack direction="row" spacing={2}>
                  <Controller
                    name="securityDeposit"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Security deposit" type="number" sx={{ flex: 1 }} slotProps={{ input: { startAdornment: '$' } }} />}
                  />
                  <Controller
                    name="rentDueDay"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Rent due day" type="number" sx={{ flex: 1 }} placeholder="1–31" />}
                  />
                </Stack>
              </FormSection>

              <Box>
                <Link
                  component="button"
                  type="button"
                  onClick={() => setNotesOpen((v) => !v)}
                  sx={{ display: 'flex', alignItems: 'center', gap: 0.5, fontSize: 12.5, fontWeight: 600 }}
                >
                  Advanced
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

      {!needsPropertyPicker && !isWaitingForRecord && (
        <>
          <Divider />
          <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'flex-end', px: 3, py: 2 }}>
            <Stack direction="row" spacing={1.5}>
              <Button variant="outlined" onClick={onClose} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
                Cancel
              </Button>
              <Button
                type="submit"
                form="unit-form"
                variant="contained"
                disabled={isPending}
                startIcon={isPending ? <CircularProgress size={15} color="inherit" /> : undefined}
              >
                {isPending ? 'Saving…' : isEditMode ? 'Save changes' : 'Add Unit'}
              </Button>
            </Stack>
          </Stack>
        </>
      )}
      </Dialog>

      {isEditMode && unitId && (
        <LeaseFormDialog
          open={leaseFormOpen}
          onClose={() => setLeaseFormOpen(false)}
          unitId={unitId}
          leaseId={activeLease?.id}
          onSaved={() => setLeaseFormOpen(false)}
        />
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
