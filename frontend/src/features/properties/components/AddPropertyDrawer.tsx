import { useRef, useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { Controller, useForm, useWatch } from 'react-hook-form';
import Accordion from '@mui/material/Accordion';
import AccordionDetails from '@mui/material/AccordionDetails';
import AccordionSummary from '@mui/material/AccordionSummary';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Collapse from '@mui/material/Collapse';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import Divider from '@mui/material/Divider';
import Drawer from '@mui/material/Drawer';
import FormControlLabel from '@mui/material/FormControlLabel';
import IconButton from '@mui/material/IconButton';
import MenuItem from '@mui/material/MenuItem';
import Radio from '@mui/material/Radio';
import RadioGroup from '@mui/material/RadioGroup';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import AddOutlined from '@mui/icons-material/AddOutlined';
import CloseOutlined from '@mui/icons-material/CloseOutlined';
import CloudUploadOutlined from '@mui/icons-material/CloudUploadOutlined';
import ElevatorOutlined from '@mui/icons-material/ElevatorOutlined';
import EvStationOutlined from '@mui/icons-material/EvStationOutlined';
import ExpandMoreOutlined from '@mui/icons-material/ExpandMoreOutlined';
import FitnessCenterOutlined from '@mui/icons-material/FitnessCenterOutlined';
import InsertDriveFileOutlined from '@mui/icons-material/InsertDriveFileOutlined';
import Inventory2Outlined from '@mui/icons-material/Inventory2Outlined';
import LocalLaundryServiceOutlined from '@mui/icons-material/LocalLaundryServiceOutlined';
import LocalParkingOutlined from '@mui/icons-material/LocalParkingOutlined';
import PetsOutlined from '@mui/icons-material/PetsOutlined';
import PoolOutlined from '@mui/icons-material/PoolOutlined';
import RemoveOutlined from '@mui/icons-material/RemoveOutlined';
import { tokens } from '@/app/tokens';
import { addPropertySchema, addPropertyDefaultValues, type AddPropertyFormValues } from '../schemas/addPropertySchema';
import { AMENITIES, COUNTRIES, PROPERTY_TYPES, US_STATES, type PropertyRow } from '../mock/propertyRows';

const AMENITY_ICONS: Record<(typeof AMENITIES)[number], React.ComponentType<{ sx?: object }>> = {
  Parking: LocalParkingOutlined,
  Laundry: LocalLaundryServiceOutlined,
  Pool: PoolOutlined,
  Elevator: ElevatorOutlined,
  'Pet-friendly': PetsOutlined,
  Gym: FitnessCenterOutlined,
  Storage: Inventory2Outlined,
  'EV charging': EvStationOutlined,
};

interface PhotoPreview {
  id: string;
  url: string;
}

interface AddPropertyDrawerProps {
  open: boolean;
  onClose: () => void;
  onSaved: (row: PropertyRow) => void;
  /** Test/demo hook: pre-fills the form so a reviewer can jump straight to a filled-but-invalid state. */
  prefillForPreview?: Partial<AddPropertyFormValues>;
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export function AddPropertyDrawer({ open, onClose, onSaved, prefillForPreview }: AddPropertyDrawerProps) {
  const {
    control,
    handleSubmit,
    setValue,
    reset,
    trigger,
    formState: { errors, isDirty, isValid },
  } = useForm<AddPropertyFormValues>({
    resolver: zodResolver(addPropertySchema),
    mode: 'onChange',
    defaultValues: prefillForPreview
      ? { ...addPropertyDefaultValues(), ...prefillForPreview }
      : addPropertyDefaultValues(),
  });

  const type = useWatch({ control, name: 'type' });
  const ownership = useWatch({ control, name: 'ownership' });
  const unitsLocked = type === 'Residential – Single Unit';

  // Errors are only ever shown for a field once it's been blurred — RHF's
  // own touchedFields doesn't reliably fire for the Controller-driven
  // fields here (ToggleButtonGroup, the units stepper), so this is set
  // explicitly by each field's onBlur instead.
  const [touched, setTouched] = useState<Partial<Record<keyof AddPropertyFormValues, boolean>>>({});
  const markTouched = (key: keyof AddPropertyFormValues) => setTouched((prev) => ({ ...prev, [key]: true }));

  const [photosOpen, setPhotosOpen] = useState(false);
  const [photos, setPhotos] = useState<PhotoPreview[]>([]);
  const [insuranceFile, setInsuranceFile] = useState<File | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [discardOpen, setDiscardOpen] = useState(false);
  const [dragOver, setDragOver] = useState(false);

  const photoInputRef = useRef<HTMLInputElement>(null);
  const insuranceInputRef = useRef<HTMLInputElement>(null);

  const hasExtraContent = photos.length > 0 || insuranceFile !== null;

  function resetEverything() {
    reset(addPropertyDefaultValues());
    photos.forEach((p) => URL.revokeObjectURL(p.url));
    setPhotos([]);
    setInsuranceFile(null);
    setPhotosOpen(false);
    setDiscardOpen(false);
    setTouched({});
  }

  function requestClose() {
    if (isDirty || hasExtraContent) {
      setDiscardOpen(true);
    } else {
      onClose();
    }
  }

  function confirmDiscard() {
    resetEverything();
    onClose();
  }

  function addPhotoFiles(fileList: FileList | null) {
    if (!fileList) return;
    const remaining = Math.max(0, 10 - photos.length);
    const next = Array.from(fileList)
      .filter((f) => f.type.startsWith('image/'))
      .slice(0, remaining)
      .map((f) => ({ id: `${Date.now()}-${Math.random()}`, url: URL.createObjectURL(f) }));
    if (next.length) setPhotos((prev) => [...prev, ...next]);
  }

  function removePhoto(id: string) {
    setPhotos((prev) => {
      const target = prev.find((p) => p.id === id);
      if (target) URL.revokeObjectURL(target.url);
      return prev.filter((p) => p.id !== id);
    });
  }

  const onValid = (values: AddPropertyFormValues) => {
    setIsSaving(true);
    // Simulated save latency — there's no real backend field for most of
    // this form yet (see mock/propertyRows.ts), so this never leaves the browser.
    setTimeout(() => {
      const row: PropertyRow = {
        id: `p-${Date.now()}`,
        name: values.name,
        address: [values.addressLine1, values.city, values.stateProvince].filter(Boolean).join(', '),
        type: values.type,
        units: values.units,
        occupancyPct: 0,
        collectedThisMonth: 0,
        status: 'Onboarding',
      };
      setIsSaving(false);
      resetEverything();
      onSaved(row);
    }, 900);
  };

  const onInvalid = () => {
    const order: (keyof AddPropertyFormValues)[] = ['name', 'type', 'addressLine1', 'city', 'stateProvince', 'postalCode', 'units'];
    order.forEach(markTouched);
    const firstBad = order.find((key) => errors[key]);
    if (firstBad) {
      document.getElementById(`field-${firstBad}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  };

  return (
    <>
      <Drawer
        anchor="right"
        open={open}
        onClose={requestClose}
        slotProps={{ paper: { sx: { width: { xs: '100%', sm: 520 }, display: 'flex', flexDirection: 'column' } } }}
      >
        <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', px: 3, py: 2.25 }}>
          <Typography variant="h4">Add Property</Typography>
          <IconButton onClick={requestClose} aria-label="Close">
            <CloseOutlined />
          </IconButton>
        </Stack>
        <Divider />

        <Box
          component="form"
          id="add-property-form"
          onSubmit={handleSubmit(onValid, onInvalid)}
          noValidate
          sx={{ flex: 1, overflowY: 'auto', px: 3, py: 3, display: 'flex', flexDirection: 'column', gap: 4 }}
        >
          {/* A — Basic info */}
          <Stack spacing={2.25}>
            <SectionLabel>Basic info</SectionLabel>

            <TextField
              id="field-name"
              label="Property name"
              required
              fullWidth
              {...control.register('name')}
              onChange={(e) => setValue('name', e.target.value, { shouldValidate: true, shouldDirty: true })}
              onBlur={() => {
                markTouched('name');
                void trigger('name');
              }}
              error={Boolean(touched.name && errors.name)}
              helperText={touched.name && errors.name?.message}
              placeholder="e.g. Willow Creek Apartments"
            />

            <Box id="field-type">
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: 'text.secondary', mb: 1 }}>
                Property type <Box component="span" sx={{ color: 'error.main' }}>*</Box>
              </Typography>
              <Controller
                name="type"
                control={control}
                render={({ field }) => (
                  <ToggleButtonGroup
                    exclusive
                    value={field.value ?? null}
                    onChange={(_e, value: AddPropertyFormValues['type'] | null) => {
                      if (!value) return;
                      field.onChange(value);
                      markTouched('type');
                      void trigger('type');
                      if (value === 'Residential – Single Unit') setValue('units', 1, { shouldValidate: true });
                    }}
                    sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 1, width: '100%' }}
                  >
                    {PROPERTY_TYPES.map((t) => (
                      <ToggleButton
                        key={t}
                        value={t}
                        sx={{
                          border: `1.5px solid ${tokens.slate[200]} !important`,
                          borderRadius: `${tokens.radiusControl}px !important`,
                          textTransform: 'none',
                          fontWeight: 600,
                          fontSize: 13,
                          color: tokens.slate[600],
                          justifyContent: 'flex-start',
                          textAlign: 'left',
                          lineHeight: 1.3,
                          py: 1,
                          '&.Mui-selected': { bgcolor: tokens.azure[50], color: tokens.azure[700], borderColor: `${tokens.azure[500]} !important` },
                          '&.Mui-selected:hover': { bgcolor: tokens.azure[50] },
                        }}
                      >
                        {t.replace(' – ', ' –​ ')}
                      </ToggleButton>
                    ))}
                  </ToggleButtonGroup>
                )}
              />
              {touched.type && errors.type && (
                <Typography sx={{ fontSize: 12, color: 'error.main', mt: 0.75 }}>{errors.type.message}</Typography>
              )}
            </Box>

            <TextField
              id="field-addressLine1"
              label="Address line 1"
              required
              fullWidth
              {...control.register('addressLine1')}
              onChange={(e) => setValue('addressLine1', e.target.value, { shouldValidate: true, shouldDirty: true })}
              onBlur={() => {
                markTouched('addressLine1');
                void trigger('addressLine1');
              }}
              error={Boolean(touched.addressLine1 && errors.addressLine1)}
              helperText={touched.addressLine1 && errors.addressLine1?.message}
              placeholder="Street address"
            />

            <TextField
              label="Apt / suite"
              fullWidth
              {...control.register('addressLine2')}
              placeholder="Unit, suite, floor (optional)"
            />

            <Stack direction="row" spacing={2}>
              <TextField
                id="field-city"
                label="City"
                required
                fullWidth
                {...control.register('city')}
                onChange={(e) => setValue('city', e.target.value, { shouldValidate: true, shouldDirty: true })}
                onBlur={() => {
                  markTouched('city');
                  void trigger('city');
                }}
                error={Boolean(touched.city && errors.city)}
                helperText={touched.city && errors.city?.message}
              />
              <Controller
                name="stateProvince"
                control={control}
                render={({ field }) => (
                  <TextField
                    id="field-stateProvince"
                    select
                    label="State / province"
                    required
                    fullWidth
                    {...field}
                    onBlur={() => {
                      field.onBlur();
                      markTouched('stateProvince');
                      void trigger('stateProvince');
                    }}
                    error={Boolean(touched.stateProvince && errors.stateProvince)}
                    helperText={touched.stateProvince && errors.stateProvince?.message}
                  >
                    <MenuItem value="">
                      <em>Select…</em>
                    </MenuItem>
                    {US_STATES.map((s) => (
                      <MenuItem key={s} value={s}>
                        {s}
                      </MenuItem>
                    ))}
                  </TextField>
                )}
              />
            </Stack>

            <Stack direction="row" spacing={2}>
              <TextField
                id="field-postalCode"
                label="Postal code"
                required
                fullWidth
                {...control.register('postalCode')}
                onChange={(e) => setValue('postalCode', e.target.value, { shouldValidate: true, shouldDirty: true })}
                onBlur={() => {
                  markTouched('postalCode');
                  void trigger('postalCode');
                }}
                error={Boolean(touched.postalCode && errors.postalCode)}
                helperText={touched.postalCode && errors.postalCode?.message}
              />
              <Controller
                name="country"
                control={control}
                render={({ field }) => (
                  <TextField select label="Country" required fullWidth {...field}>
                    {COUNTRIES.map((c) => (
                      <MenuItem key={c} value={c}>
                        {c}
                      </MenuItem>
                    ))}
                  </TextField>
                )}
              />
            </Stack>
          </Stack>

          {/* B — Details */}
          <Stack spacing={2.25}>
            <SectionLabel>Details</SectionLabel>

            <Box>
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: 'text.secondary', mb: 1 }}>
                Number of units <Box component="span" sx={{ color: 'error.main' }}>*</Box>
              </Typography>
              <Controller
                name="units"
                control={control}
                render={({ field }) => (
                  <Tooltip title={unitsLocked ? 'Single-unit properties always have 1 unit.' : ''} placement="right">
                    <span style={{ display: 'inline-block', width: 160 }}>
                      <TextField
                        value={field.value}
                        disabled={unitsLocked}
                        onChange={(e) => {
                          const n = parseInt(e.target.value.replace(/\D/g, ''), 10);
                          field.onChange(Number.isNaN(n) ? 0 : n);
                        }}
                        onBlur={() => {
                          if (!field.value || field.value < 1) field.onChange(1);
                          field.onBlur();
                          markTouched('units');
                          void trigger('units');
                        }}
                        slotProps={{
                          input: {
                            startAdornment: (
                              <IconButton
                                size="small"
                                disabled={unitsLocked}
                                onClick={() => field.onChange(Math.max(1, (field.value || 1) - 1))}
                                aria-label="Decrease units"
                              >
                                <RemoveOutlined fontSize="small" />
                              </IconButton>
                            ),
                            endAdornment: (
                              <IconButton
                                size="small"
                                disabled={unitsLocked}
                                onClick={() => field.onChange((field.value || 1) + 1)}
                                aria-label="Increase units"
                              >
                                <AddOutlined fontSize="small" />
                              </IconButton>
                            ),
                            sx: { pl: 0.5, pr: 0.5 },
                          },
                          htmlInput: { style: { textAlign: 'center', fontFamily: tokens.fontMono, fontWeight: 600 } },
                        }}
                        fullWidth
                      />
                    </span>
                  </Tooltip>
                )}
              />
            </Box>

            <Box>
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: 'text.secondary', mb: 1 }}>
                Ownership type <Box component="span" sx={{ color: tokens.slate[400], fontWeight: 500 }}>(optional)</Box>
              </Typography>
              <Controller
                name="ownership"
                control={control}
                render={({ field }) => (
                  <RadioGroup row {...field} value={field.value ?? ''}>
                    <FormControlLabel value="owned" control={<Radio size="small" />} label="Owned" />
                    <FormControlLabel value="managed" control={<Radio size="small" />} label="Managed on behalf of owner" />
                  </RadioGroup>
                )}
              />
            </Box>

            <Collapse in={ownership === 'managed'}>
              <TextField
                label="Owner name / entity"
                fullWidth
                {...control.register('ownerName')}
                placeholder="e.g. Alvarez Family Trust"
              />
            </Collapse>

            <Stack direction="row" spacing={2}>
              <TextField
                label="Year built"
                fullWidth
                {...control.register('yearBuilt')}
                placeholder="(optional)"
                slotProps={{ htmlInput: { inputMode: 'numeric', maxLength: 4, style: { fontFamily: tokens.fontMono } } }}
              />
              <TextField
                label="Onboard date"
                type="date"
                fullWidth
                {...control.register('onboardDate')}
                slotProps={{ inputLabel: { shrink: true } }}
              />
            </Stack>

            <Box>
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: 'text.secondary', mb: 1 }}>
                Amenities <Box component="span" sx={{ color: tokens.slate[400], fontWeight: 500 }}>(optional)</Box>
              </Typography>
              <Controller
                name="amenities"
                control={control}
                render={({ field }) => (
                  <Stack direction="row" flexWrap="wrap" gap={1}>
                    {AMENITIES.map((a) => {
                      const on = field.value.includes(a);
                      const Icon = AMENITY_ICONS[a];
                      return (
                        <Chip
                          key={a}
                          icon={<Icon sx={{ fontSize: 15 }} />}
                          label={a}
                          clickable
                          onClick={() => field.onChange(on ? field.value.filter((v) => v !== a) : [...field.value, a])}
                          variant={on ? 'filled' : 'outlined'}
                          color={on ? 'primary' : 'default'}
                          size="small"
                        />
                      );
                    })}
                  </Stack>
                )}
              />
            </Box>
          </Stack>

          {/* C — Photos & documents */}
          <Accordion
            disableGutters
            elevation={0}
            expanded={photosOpen}
            onChange={(_e, exp) => setPhotosOpen(exp)}
            sx={{ border: `1px solid ${tokens.slate[200]}`, borderRadius: `${tokens.radiusCard}px !important`, '&:before': { display: 'none' } }}
          >
            <AccordionSummary expandIcon={<ExpandMoreOutlined />}>
              <Typography sx={{ fontSize: 13, fontWeight: 800, letterSpacing: '0.04em', textTransform: 'uppercase', color: tokens.slate[600] }}>
                Add photos &amp; documents{' '}
                <Box component="span" sx={{ fontWeight: 600, textTransform: 'none', letterSpacing: 0, color: tokens.slate[400] }}>
                  (optional)
                </Box>
              </Typography>
            </AccordionSummary>
            <AccordionDetails sx={{ display: 'flex', flexDirection: 'column', gap: 2.5, pb: 2.5 }}>
              <Box>
                <Typography sx={{ fontSize: 13, fontWeight: 600, color: 'text.secondary', mb: 1 }}>Property photos</Typography>
                <Box
                  onClick={() => photoInputRef.current?.click()}
                  onDragOver={(e) => {
                    e.preventDefault();
                    setDragOver(true);
                  }}
                  onDragLeave={() => setDragOver(false)}
                  onDrop={(e) => {
                    e.preventDefault();
                    setDragOver(false);
                    addPhotoFiles(e.dataTransfer.files);
                  }}
                  sx={{
                    border: `1.5px dashed ${dragOver ? tokens.azure[500] : tokens.slate[200]}`,
                    bgcolor: dragOver ? tokens.azure[50] : tokens.slate[50],
                    borderRadius: `${tokens.radiusCard}px`,
                    p: 3,
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    gap: 0.5,
                    textAlign: 'center',
                    cursor: 'pointer',
                  }}
                >
                  <CloudUploadOutlined sx={{ fontSize: 26, color: tokens.slate[400] }} />
                  <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>Drag photos here, or click to browse</Typography>
                  <Typography sx={{ fontSize: 11.5, color: tokens.slate[400] }}>PNG or JPG, up to 10 photos</Typography>
                </Box>
                <input
                  ref={photoInputRef}
                  type="file"
                  accept="image/*"
                  multiple
                  hidden
                  onChange={(e) => {
                    addPhotoFiles(e.target.files);
                    e.target.value = '';
                  }}
                />
                {photos.length > 0 && (
                  <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(72px, 1fr))', gap: 1, mt: 1.5 }}>
                    {photos.map((p) => (
                      <Box key={p.id} sx={{ position: 'relative', aspectRatio: '1', borderRadius: 1.5, overflow: 'hidden', border: `1px solid ${tokens.slate[200]}` }}>
                        <Box component="img" src={p.url} alt="" sx={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }} />
                        <IconButton
                          size="small"
                          onClick={() => removePhoto(p.id)}
                          aria-label="Remove photo"
                          sx={{ position: 'absolute', top: 3, right: 3, width: 20, height: 20, bgcolor: 'rgba(15,17,22,0.65)', color: '#fff', '&:hover': { bgcolor: 'rgba(15,17,22,0.85)' } }}
                        >
                          <CloseOutlined sx={{ fontSize: 13 }} />
                        </IconButton>
                      </Box>
                    ))}
                  </Box>
                )}
              </Box>

              <Box>
                <Typography sx={{ fontSize: 13, fontWeight: 600, color: 'text.secondary', mb: 1 }}>Insurance document</Typography>
                {!insuranceFile ? (
                  <Box
                    onClick={() => insuranceInputRef.current?.click()}
                    sx={{
                      border: `1.5px dashed ${tokens.slate[200]}`,
                      bgcolor: tokens.slate[50],
                      borderRadius: `${tokens.radiusCard}px`,
                      p: 2,
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'center',
                      gap: 0.5,
                      cursor: 'pointer',
                    }}
                  >
                    <InsertDriveFileOutlined sx={{ fontSize: 22, color: tokens.slate[400] }} />
                    <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>Upload PDF or image</Typography>
                  </Box>
                ) : (
                  <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center', border: `1px solid ${tokens.slate[200]}`, borderRadius: `${tokens.radiusControl}px`, p: 1.25, bgcolor: tokens.slate[50] }}>
                    <Box sx={{ width: 32, height: 32, borderRadius: 1.5, bgcolor: tokens.azure[50], color: tokens.azure[600], display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                      <InsertDriveFileOutlined sx={{ fontSize: 16 }} />
                    </Box>
                    <Box sx={{ flex: 1, minWidth: 0 }}>
                      <Typography sx={{ fontSize: 13, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{insuranceFile.name}</Typography>
                      <Typography sx={{ fontSize: 11.5, color: tokens.slate[500], fontFamily: tokens.fontMono }}>{formatFileSize(insuranceFile.size)}</Typography>
                    </Box>
                    <Button size="small" onClick={() => setInsuranceFile(null)}>
                      Remove
                    </Button>
                  </Stack>
                )}
                <input
                  ref={insuranceInputRef}
                  type="file"
                  accept="application/pdf,image/*"
                  hidden
                  onChange={(e) => {
                    if (e.target.files?.[0]) setInsuranceFile(e.target.files[0]);
                  }}
                />
              </Box>
            </AccordionDetails>
          </Accordion>

          {/* D — Notes */}
          <Stack spacing={1}>
            <SectionLabel>Notes</SectionLabel>
            <TextField
              multiline
              minRows={3}
              fullWidth
              {...control.register('notes')}
              placeholder="Internal notes — not visible to tenants"
            />
          </Stack>
        </Box>

        <Divider />
        <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center', px: 3, py: 2 }}>
          <Typography sx={{ fontSize: 12, color: tokens.slate[500], flex: 1 }}>You&apos;ll be able to add units next</Typography>
          <Button variant="outlined" onClick={requestClose} disabled={isSaving} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
            Cancel
          </Button>
          <Button
            type="submit"
            form="add-property-form"
            variant="contained"
            disabled={isSaving || !isValid}
            startIcon={isSaving ? <CircularProgress size={15} color="inherit" /> : undefined}
          >
            {isSaving ? 'Saving…' : 'Save Property'}
          </Button>
        </Stack>
      </Drawer>

      <Dialog open={discardOpen} onClose={() => setDiscardOpen(false)} maxWidth="xs">
        <DialogTitle sx={{ fontSize: 16 }}>Discard changes?</DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>You&apos;ll lose what you entered for this property.</Typography>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2.5 }}>
          <Button onClick={() => setDiscardOpen(false)}>Keep editing</Button>
          <Button onClick={confirmDiscard} color="error" variant="contained">
            Discard
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <Typography sx={{ fontSize: 13, fontWeight: 800, letterSpacing: '0.04em', textTransform: 'uppercase', color: tokens.slate[600] }}>
      {children}
    </Typography>
  );
}
