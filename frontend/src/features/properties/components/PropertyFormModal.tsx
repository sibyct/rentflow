import { forwardRef, useEffect, useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm, useWatch, type FieldErrors } from 'react-hook-form';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import Divider from '@mui/material/Divider';
import IconButton from '@mui/material/IconButton';
import Slide from '@mui/material/Slide';
import type { TransitionProps } from '@mui/material/transitions';
import Stack from '@mui/material/Stack';
import Step from '@mui/material/Step';
import StepButton from '@mui/material/StepButton';
import Stepper from '@mui/material/Stepper';
import Typography from '@mui/material/Typography';
import ArchiveOutlined from '@mui/icons-material/ArchiveOutlined';
import CheckCircleOutlined from '@mui/icons-material/CheckCircleOutlined';
import CloseOutlined from '@mui/icons-material/CloseOutlined';
import UnarchiveOutlined from '@mui/icons-material/UnarchiveOutlined';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import { LoadingSpinner } from '@/shared/components';
import { BasicInfoStep } from './steps/BasicInfoStep';
import { DetailsStep } from './steps/DetailsStep';
import { AmenitiesMediaStep, type UploadedFile } from './steps/AmenitiesMediaStep';
import { NotesStep } from './steps/NotesStep';
import { toFormValues } from '../api/propertiesApi';
import { useCreateProperty, useProperty, useUpdateProperty, useUpdatePropertyStatus } from '../hooks/usePropertiesQueries';
import { addPropertySchema, addPropertyDefaultValues, stepSchemas, type AddPropertyFormValues } from '../schemas/addPropertySchema';
import type { PropertyRow, PropertyRowStatus } from '../mock/propertyRows';

const STEP_DEFS = [
  { key: 'basics', label: 'Basic Info' },
  { key: 'details', label: 'Details' },
  { key: 'amenities', label: 'Amenities & Media' },
  { key: 'notes', label: 'Notes' },
] as const;
type StepKey = (typeof STEP_DEFS)[number]['key'];

const STEP_REQUIRED_FIELDS: Record<StepKey, (keyof AddPropertyFormValues)[]> = {
  basics: ['name', 'type', 'fullAddress'],
  details: ['units'],
  amenities: [],
  notes: [],
};

const Transition = forwardRef(function Transition(
  props: TransitionProps & { children: React.ReactElement },
  ref: React.Ref<unknown>,
) {
  return <Slide direction="up" ref={ref} {...props} />;
});

interface PropertyFormModalProps {
  open: boolean;
  onClose: () => void;
  /**
   * Editing an existing property when set; creating a new one when
   * omitted. This alone is the create/edit "mode" — deliberately not a
   * separate `mode` prop, since two flags that must always agree with
   * each other (propertyId set <=> mode 'edit') are just a way to let
   * them quietly disagree later.
   */
  propertyId?: string;
  /** Fires once, when the whole flow (including create's post-save "Add units" screen) is done. */
  onSaved: (row: PropertyRow, options: { addUnitsNow: boolean; isEdit: boolean }) => void;
  /** Edit mode only: fires after the header's Archive/Restore action succeeds, so the caller can toast. */
  onArchiveToggled?: (message: string) => void;
  /** Which step to open on — e.g. the detail page's "+ Add" (amenities) and "+ Add note" shortcuts jump straight there instead of always starting at step 1. Defaults to 0. */
  initialStepIndex?: number;
}

export function PropertyFormModal({ open, onClose, propertyId, onSaved, onArchiveToggled, initialStepIndex = 0 }: PropertyFormModalProps) {
  const isEditMode = Boolean(propertyId);
  const { data: existingProperty, isLoading: isLoadingProperty } = useProperty(open ? propertyId : undefined);

  const {
    control,
    handleSubmit,
    setValue,
    setError,
    reset,
    trigger,
    formState: { errors, isDirty: formIsDirty },
  } = useForm<AddPropertyFormValues>({
    resolver: zodResolver(addPropertySchema),
    mode: 'onChange',
    defaultValues: addPropertyDefaultValues(),
  });

  const watchedValues = useWatch({ control });
  const type = watchedValues.type;
  const ownership = watchedValues.ownership;
  const unitsLocked = type === 'Residential – Single Unit';

  // Errors only ever show for a field once it's been blurred — RHF's own
  // touchedFields doesn't reliably fire for the Controller-driven fields
  // here (ToggleButtonGroup, the units stepper), so this is set
  // explicitly by each field's onBlur instead.
  const [touched, setTouched] = useState<Partial<Record<keyof AddPropertyFormValues, boolean>>>({});
  const markTouched = (key: keyof AddPropertyFormValues) => setTouched((prev) => ({ ...prev, [key]: true }));

  const [activeStepIndex, setActiveStepIndex] = useState(initialStepIndex);

  // (Re)initializes the form every time the modal opens: blank defaults
  // for create, or the fetched record for edit — reset() also becomes
  // formState.isDirty's new comparison baseline, which is what makes
  // "close without changes" not falsely prompt a discard-confirmation
  // for edit mode (every field starts non-empty there). Also jumps to
  // whichever step the caller asked to open on (e.g. the detail page's
  // "+ Add note" shortcut opens straight to Notes).
  useEffect(() => {
    if (!open) return;
    if (isEditMode) {
      if (existingProperty) reset(toFormValues(existingProperty));
    } else {
      reset(addPropertyDefaultValues());
    }
    setActiveStepIndex(initialStepIndex);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, isEditMode, existingProperty]);
  const [phase, setPhase] = useState<'form' | 'success'>('form');
  const [savedRow, setSavedRow] = useState<PropertyRow | null>(null);

  const [files, setFiles] = useState<UploadedFile[]>([]);
  const [discardOpen, setDiscardOpen] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const createProperty = useCreateProperty();
  const updateProperty = useUpdateProperty();
  const updateStatus = useUpdatePropertyStatus();
  const isSaving = isEditMode ? updateProperty.isPending : createProperty.isPending;

  const [archiveConfirmOpen, setArchiveConfirmOpen] = useState(false);

  const currentStepKey: StepKey = (STEP_DEFS[activeStepIndex] ?? STEP_DEFS[0]).key;
  const isLastStep = activeStepIndex === STEP_DEFS.length - 1;
  const currentStepValid = stepSchemas[currentStepKey].safeParse(watchedValues).success;

  // create: a step is reachable only once every step before it validates
  // — the same rule handleNext already enforces one step at a time, just
  // checked transitively so clicking a step number can't skip past it.
  // edit: every step is reachable immediately — the record's already
  // complete, there's no "not there yet" to gate.
  function isStepReachable(index: number): boolean {
    if (isEditMode) return true;
    for (let i = 0; i < index; i++) {
      const step = STEP_DEFS[i];
      if (!step || !stepSchemas[step.key].safeParse(watchedValues).success) return false;
    }
    return true;
  }

  function stepIsComplete(index: number): boolean {
    // edit: never show a "completed" checkmark — every step already had
    // data before the user did anything, so a checkmark would read as
    // "you just finished this," which isn't true.
    if (isEditMode) return false;
    if (index >= activeStepIndex) return false;
    const step = STEP_DEFS[index];
    if (!step) return false;
    return stepSchemas[step.key].safeParse(watchedValues).success;
  }

  const isDirty = formIsDirty || files.length > 0;

  function resetEverything() {
    reset(addPropertyDefaultValues());
    files.forEach((f) => f.previewUrl && URL.revokeObjectURL(f.previewUrl));
    setFiles([]);
    setTouched({});
    setActiveStepIndex(0);
    setPhase('form');
    setSavedRow(null);
    setDiscardOpen(false);
    setArchiveConfirmOpen(false);
    setSubmitError(null);
    createProperty.reset();
    updateProperty.reset();
  }

  function requestClose() {
    if (phase === 'success') {
      // Already saved — closing here is the same as "I'll do this
      // later", not a discard of anything pending.
      finish(false);
      return;
    }
    // create: a blank-ish draft that was never saved is low-stakes to
    // throw away — closing just discards it, no confirmation.
    // edit: the record had real, previously-saved data before this
    // session started, so an unsaved edit disappearing silently would
    // be a much more surprising loss — confirm first.
    if (isEditMode && isDirty) {
      setDiscardOpen(true);
      return;
    }
    resetEverything();
    onClose();
  }

  function confirmDiscard() {
    resetEverything();
    onClose();
  }

  function toggleArchive() {
    if (!propertyId || !existingProperty) return;
    const nextStatus: PropertyRowStatus = existingProperty.status === 'Archived' ? 'Active' : 'Archived';
    updateStatus.mutate(
      { ids: [propertyId], status: nextStatus },
      {
        onSuccess: () => {
          setArchiveConfirmOpen(false);
          onArchiveToggled?.(`${nextStatus === 'Archived' ? 'Archived' : 'Restored'} ${existingProperty.name}`);
          resetEverything();
          onClose();
        },
        onError: (error) => {
          setArchiveConfirmOpen(false);
          setSubmitError(error instanceof Error ? error.message : 'Could not update the property status. Please try again.');
        },
      },
    );
  }

  function addFiles(fileList: FileList | null) {
    if (!fileList) return;
    const remaining = Math.max(0, 10 - files.length);
    const next = Array.from(fileList)
      .slice(0, remaining)
      .map((f) => ({
        id: `${Date.now()}-${Math.random()}`,
        file: f,
        previewUrl: f.type.startsWith('image/') ? URL.createObjectURL(f) : undefined,
      }));
    if (next.length) setFiles((prev) => [...prev, ...next]);
  }

  function removeFile(id: string) {
    setFiles((prev) => {
      const target = prev.find((f) => f.id === id);
      if (target?.previewUrl) URL.revokeObjectURL(target.previewUrl);
      return prev.filter((f) => f.id !== id);
    });
  }

  function handleNext() {
    STEP_REQUIRED_FIELDS[currentStepKey].forEach(markTouched);
    void trigger(STEP_REQUIRED_FIELDS[currentStepKey]);
    if (!currentStepValid) return;
    setActiveStepIndex((i) => Math.min(i + 1, STEP_DEFS.length - 1));
  }

  function handleBack() {
    setActiveStepIndex((i) => Math.max(i - 1, 0));
  }

  const onValid = (values: AddPropertyFormValues) => {
    setSubmitError(null);

    if (isEditMode && propertyId) {
      updateProperty.mutate(
        { id: propertyId, values },
        {
          onSuccess: (detail) => {
            onSaved(detail, { addUnitsNow: false, isEdit: true });
            resetEverything();
            onClose();
          },
          onError: (error) => {
            setSubmitError(error instanceof Error ? error.message : 'Could not save your changes. Please try again.');
          },
        },
      );
      return;
    }

    createProperty.mutate(values, {
      onSuccess: (row) => {
        setSavedRow(row);
        setPhase('success');
      },
      onError: (error) => {
        // A duplicate-address rejection from the service layer (see
        // backend/internal/service/property_service.go) names the
        // address_line1 field — surface it on the step-1 address field
        // specifically, since that's the one field the user can fix.
        if (error instanceof ApiError && error.validationDetails()?.some((d) => d.field === 'address_line1')) {
          setError('fullAddress', { message: 'You already have a property at this address.' });
          setActiveStepIndex(0);
          return;
        }
        setSubmitError(error instanceof Error ? error.message : 'Could not save the property. Please try again.');
      },
    });
  };

  const onInvalid = (formErrors: FieldErrors<AddPropertyFormValues>) => {
    // Edit mode's persistent "Save changes" submits from wherever the
    // user currently is, not just the last step — so a validation
    // failure can land on a step that isn't the one currently shown.
    // Jump to the first step that actually has an error, so it's never
    // silently invisible.
    const erroredStepIndex = STEP_DEFS.findIndex((step) =>
      STEP_REQUIRED_FIELDS[step.key].some((field) => field in formErrors),
    );
    const targetIndex = erroredStepIndex === -1 ? activeStepIndex : erroredStepIndex;
    if (targetIndex !== activeStepIndex) setActiveStepIndex(targetIndex);
    const targetKey = (STEP_DEFS[targetIndex] ?? STEP_DEFS[0]).key;
    STEP_REQUIRED_FIELDS[targetKey].forEach(markTouched);
  };

  function finish(addUnitsNow: boolean) {
    if (savedRow) onSaved(savedRow, { addUnitsNow, isEdit: false });
    resetEverything();
    onClose();
  }

  const isWaitingForRecord = isEditMode && (isLoadingProperty || !existingProperty);
  const title = isEditMode ? 'Edit Property' : phase === 'success' ? 'Property added' : 'Add Property';

  return (
    <>
      <Dialog
        fullScreen
        open={open}
        onClose={requestClose}
        slots={{ transition: Transition }}
        aria-labelledby="property-form-title"
      >
        <Stack direction="row" sx={{ alignItems: 'flex-start', justifyContent: 'space-between', px: 3, py: 2 }}>
          <Box>
            <Typography id="property-form-title" variant="h4" sx={{ fontSize: 18 }}>
              {title}
            </Typography>
            {/* Identifies which record this is, for anyone with more than
                one property's edit modal reachable across tabs/sessions. */}
            {isEditMode && existingProperty && !isWaitingForRecord && (
              <Typography sx={{ fontSize: 12.5, color: tokens.slate[500], mt: 0.25 }}>{existingProperty.name}</Typography>
            )}
          </Box>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
            {isEditMode && !isWaitingForRecord && phase === 'form' && (
              <Button
                size="small"
                variant="outlined"
                color="error"
                startIcon={existingProperty?.status === 'Archived' ? <UnarchiveOutlined fontSize="small" /> : <ArchiveOutlined fontSize="small" />}
                onClick={() => setArchiveConfirmOpen(true)}
                sx={{ borderColor: tokens.slate[300] }}
              >
                {existingProperty?.status === 'Archived' ? 'Restore' : 'Archive'}
              </Button>
            )}
            <IconButton onClick={requestClose} aria-label="Close">
              <CloseOutlined />
            </IconButton>
          </Stack>
        </Stack>
        <Divider />

        {phase === 'form' && !isWaitingForRecord && (
          <Box sx={{ px: 3, py: 2.5, borderBottom: `1px solid ${tokens.slate[100]}` }}>
            <Stepper activeStep={activeStepIndex} sx={{ maxWidth: 720, mx: 'auto' }}>
              {STEP_DEFS.map((step, index) => (
                <Step key={step.key} completed={stepIsComplete(index)}>
                  <StepButton onClick={() => setActiveStepIndex(index)} disabled={!isStepReachable(index)}>
                    {step.label}
                  </StepButton>
                </Step>
              ))}
            </Stepper>
          </Box>
        )}

        <DialogContent sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', overflowX: 'hidden' }}>
          {isWaitingForRecord ? (
            <Box sx={{ py: 8 }}>
              <LoadingSpinner label="Loading property…" />
            </Box>
          ) : (
            <Box
              component={phase === 'form' ? 'form' : 'div'}
              id="property-form"
              {...(phase === 'form' ? { onSubmit: handleSubmit(onValid, onInvalid), noValidate: true } : {})}
              sx={{ width: '100%', maxWidth: 640, py: 3 }}
            >
              {phase === 'form' ? (
                <>
                  {submitError && (
                    <Alert severity="error" sx={{ mb: 2.5 }} onClose={() => setSubmitError(null)}>
                      {submitError}
                    </Alert>
                  )}
                  {currentStepKey === 'basics' && (
                    <BasicInfoStep control={control} setValue={setValue} trigger={trigger} errors={errors} touched={touched} markTouched={markTouched} />
                  )}
                  {currentStepKey === 'details' && (
                    <DetailsStep
                      control={control}
                      trigger={trigger}
                      errors={errors}
                      touched={touched}
                      markTouched={markTouched}
                      unitsLocked={unitsLocked}
                      ownership={ownership}
                    />
                  )}
                  {currentStepKey === 'amenities' && (
                    <AmenitiesMediaStep control={control} files={files} onAddFiles={addFiles} onRemoveFile={removeFile} />
                  )}
                  {currentStepKey === 'notes' && <NotesStep control={control} />}
                </>
              ) : (
                <SuccessScreen row={savedRow} onAddUnits={() => finish(true)} onLater={() => finish(false)} />
              )}
            </Box>
          )}
        </DialogContent>

        {phase === 'form' && !isWaitingForRecord && (
          <>
            <Divider />
            <Stack direction="row" sx={{ alignItems: 'center', px: 3, py: 2 }}>
              <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>
                Step {activeStepIndex + 1} of {STEP_DEFS.length}
              </Typography>
              <Stack direction="row" spacing={1.5} sx={{ ml: 'auto' }}>
                {activeStepIndex > 0 && (
                  <Button variant="outlined" onClick={handleBack} disabled={isSaving} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
                    Back
                  </Button>
                )}
                {isEditMode ? (
                  <>
                    {/* Edit mode: the record already exists and is
                        already valid, so committing a change made on
                        step 1 shouldn't require clicking through every
                        later step first — Save changes is always here,
                        on every step. Next stays too, purely for
                        browsing the rest of the record. */}
                    {!isLastStep && (
                      <Button variant="outlined" onClick={handleNext} disabled={!currentStepValid} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
                        Next
                      </Button>
                    )}
                    <Button
                      type="submit"
                      form="property-form"
                      variant="contained"
                      disabled={isSaving}
                      startIcon={isSaving ? <CircularProgress size={15} color="inherit" /> : undefined}
                    >
                      {isSaving ? 'Saving…' : 'Save changes'}
                    </Button>
                  </>
                ) : !isLastStep ? (
                  <Button variant="contained" onClick={handleNext} disabled={!currentStepValid}>
                    Next
                  </Button>
                ) : (
                  <Button
                    type="submit"
                    form="property-form"
                    variant="contained"
                    disabled={isSaving}
                    startIcon={isSaving ? <CircularProgress size={15} color="inherit" /> : undefined}
                  >
                    {isSaving ? 'Saving…' : 'Save Property'}
                  </Button>
                )}
              </Stack>
            </Stack>
          </>
        )}
      </Dialog>

      {/* Only ever opened in edit mode (see requestClose) — create's
          "just discard, no confirmation" path never sets this. */}
      <Dialog open={discardOpen} onClose={() => setDiscardOpen(false)} maxWidth="xs">
        <DialogTitle sx={{ fontSize: 16 }}>Discard unsaved changes?</DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>
            {existingProperty?.name ?? 'This property'} will revert to its last saved version.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2.5 }}>
          <Button onClick={() => setDiscardOpen(false)}>Keep editing</Button>
          <Button onClick={confirmDiscard} color="error" variant="contained">
            Discard
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={archiveConfirmOpen} onClose={() => setArchiveConfirmOpen(false)} maxWidth="xs">
        <DialogTitle sx={{ fontSize: 16 }}>
          {existingProperty?.status === 'Archived' ? 'Restore this property?' : 'Archive this property?'}
        </DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>
            {existingProperty?.status === 'Archived'
              ? `${existingProperty?.name} will move back to your active properties.`
              : `${existingProperty?.name} will move out of your active properties. You can restore it later.`}
          </Typography>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2.5 }}>
          <Button onClick={() => setArchiveConfirmOpen(false)}>Cancel</Button>
          <Button onClick={toggleArchive} color="error" variant="contained" disabled={updateStatus.isPending}>
            {existingProperty?.status === 'Archived' ? 'Restore' : 'Archive'}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}

function SuccessScreen({ row, onAddUnits, onLater }: { row: PropertyRow | null; onAddUnits: () => void; onLater: () => void }) {
  return (
    <Stack sx={{ alignItems: 'center', textAlign: 'center', py: 6, gap: 2 }}>
      <Box sx={{ width: 64, height: 64, borderRadius: '50%', bgcolor: tokens.azure[50], color: tokens.azure[600], display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <CheckCircleOutlined sx={{ fontSize: 32 }} />
      </Box>
      <Typography variant="h4" sx={{ fontSize: 20 }}>
        {row?.name ?? 'Property'} added
      </Typography>
      <Typography sx={{ fontSize: 13.5, color: tokens.slate[600], maxWidth: 380 }}>
        You can set up individual units — floor plans, rent, and lease terms — right away, or come back to it later.
      </Typography>
      <Stack direction="row" spacing={1.5} sx={{ mt: 1 }}>
        <Button variant="outlined" onClick={onLater} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
          I&apos;ll do this later
        </Button>
        <Button variant="contained" onClick={onAddUnits}>
          Add units now
        </Button>
      </Stack>
    </Stack>
  );
}
