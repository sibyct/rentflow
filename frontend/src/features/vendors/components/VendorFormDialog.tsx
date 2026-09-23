import { forwardRef, useEffect, useState, type ReactNode } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogContent from '@mui/material/DialogContent';
import Divider from '@mui/material/Divider';
import FormControlLabel from '@mui/material/FormControlLabel';
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
import type { PropertyRow } from '@/features/properties/mock/propertyRows';
import { FileUpload } from '@/shared/components';
import { toFormValues } from '../api/vendorsApi';
import { useCreateVendor, useUpdateVendor, useVendor, useVendorPropertiesServed } from '../hooks/useVendorsQueries';
import { vendorDefaultValues, vendorSchema, type VendorFormValues } from '../schemas/vendorSchema';
import { VENDOR_PAYMENT_TERMS, VENDOR_PAYMENT_TERMS_LABELS, VENDOR_RATE_TYPES, VENDOR_RATE_TYPE_LABELS, WORK_ORDER_CATEGORIES, WORK_ORDER_CATEGORY_LABELS } from '../types';

const Transition = forwardRef(function Transition(
  props: TransitionProps & { children: React.ReactElement },
  ref: React.Ref<unknown>,
) {
  return <Slide direction="up" ref={ref} {...props} />;
});

interface VendorFormDialogProps {
  open: boolean;
  onClose: () => void;
  properties: PropertyRow[];
  /** Present = edit an existing vendor; absent = create a new one. */
  vendorId?: string;
  onSaved: () => void;
}

/** Single-vendor add/edit popover — same Dialog chrome as PropertyFormModal/UnitFormDialog/LeaseFormDialog. */
export function VendorFormDialog({ open, onClose, properties, vendorId, onSaved }: VendorFormDialogProps) {
  const isEditMode = Boolean(vendorId);

  const { data: existingVendor, isLoading: isLoadingVendor } = useVendor(vendorId);
  const { data: propertiesServed, isLoading: isLoadingPropertiesServed } = useVendorPropertiesServed(vendorId, {
    enabled: Boolean(existingVendor) && !existingVendor?.servesAllProperties,
  });
  const createVendor = useCreateVendor();
  const updateVendor = useUpdateVendor();
  const [submitError, setSubmitError] = useState<string | null>(null);
  // Names of just-uploaded files this session — an existing attachment
  // loaded from the record shows a generic label instead (see FileUpload).
  const [coiName, setCoiName] = useState('');
  const [taxDocName, setTaxDocName] = useState('');

  const {
    control,
    handleSubmit,
    reset,
    watch,
    formState: { errors },
  } = useForm<VendorFormValues>({
    resolver: zodResolver(vendorSchema),
    defaultValues: vendorDefaultValues(),
  });

  const servesAllProperties = watch('servesAllProperties');
  const needsPropertiesServed = isEditMode && !existingVendor?.servesAllProperties;
  const isWaitingForRecord = isEditMode && (isLoadingVendor || !existingVendor || (needsPropertiesServed && isLoadingPropertiesServed));

  useEffect(() => {
    if (!open) return;
    if (isEditMode) {
      if (existingVendor && (existingVendor.servesAllProperties || propertiesServed)) {
        reset(toFormValues(existingVendor, propertiesServed ?? []));
      }
    } else {
      reset(vendorDefaultValues());
    }
    setSubmitError(null);
    setCoiName('');
    setTaxDocName('');
  }, [open, isEditMode, existingVendor, propertiesServed, reset]);

  const isPending = createVendor.isPending || updateVendor.isPending;
  const title = isEditMode ? 'Edit Vendor' : 'Add Vendor';

  function onSubmit(values: VendorFormValues) {
    setSubmitError(null);
    const onError = (err: unknown) => setSubmitError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.');

    if (isEditMode && vendorId) {
      updateVendor.mutate({ id: vendorId, values }, { onSuccess: onSaved, onError });
    } else {
      createVendor.mutate(values, { onSuccess: onSaved, onError });
    }
  }

  return (
    <Dialog fullScreen open={open} onClose={onClose} slots={{ transition: Transition }} aria-labelledby="vendor-form-title">
      <Stack direction="row" sx={{ alignItems: 'flex-start', justifyContent: 'space-between', px: 3, py: 2 }}>
        <Typography id="vendor-form-title" variant="h4" sx={{ fontSize: 18 }}>
          {title}
        </Typography>
        <IconButton onClick={onClose} aria-label="Close">
          <CloseOutlined />
        </IconButton>
      </Stack>
      <Divider />

      <DialogContent sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', overflowX: 'hidden' }}>
        {isWaitingForRecord ? (
          <Box sx={{ py: 8 }}>
            <Typography sx={{ fontSize: 13.5, color: tokens.slate[500] }}>Loading vendor…</Typography>
          </Box>
        ) : (
          <Box component="form" id="vendor-form" onSubmit={handleSubmit(onSubmit)} sx={{ width: '100%', maxWidth: 640, py: 3 }}>
            <Stack spacing={3}>
              {submitError && <Alert severity="error">{submitError}</Alert>}

              <FormSection title="Basic info">
                <Controller
                  name="companyName"
                  control={control}
                  render={({ field }) => (
                    <TextField {...field} label="Company name" required fullWidth error={Boolean(errors.companyName)} helperText={errors.companyName?.message} />
                  )}
                />
                <Controller
                  name="categories"
                  control={control}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      select
                      label="Categories"
                      required
                      fullWidth
                      error={Boolean(errors.categories)}
                      helperText={errors.categories?.message}
                      slotProps={{
                        select: {
                          multiple: true,
                          renderValue: (selected) => (
                            <Stack direction="row" spacing={0.5} sx={{ flexWrap: 'wrap', rowGap: 0.5 }}>
                              {(selected as string[]).map((c) => (
                                <Chip key={c} label={WORK_ORDER_CATEGORY_LABELS[c as keyof typeof WORK_ORDER_CATEGORY_LABELS]} size="small" />
                              ))}
                            </Stack>
                          ),
                        },
                      }}
                    >
                      {WORK_ORDER_CATEGORIES.map((c) => (
                        <MenuItem key={c} value={c}>
                          {WORK_ORDER_CATEGORY_LABELS[c]}
                        </MenuItem>
                      ))}
                    </TextField>
                  )}
                />
                <Stack direction="row" spacing={2}>
                  <Controller name="contactPerson" control={control} render={({ field }) => <TextField {...field} label="Contact person" fullWidth />} />
                  <Controller name="phone" control={control} render={({ field }) => <TextField {...field} label="Phone" fullWidth />} />
                </Stack>
                <Controller name="email" control={control} render={({ field }) => <TextField {...field} label="Email" fullWidth />} />
                <Controller name="address" control={control} render={({ field }) => <TextField {...field} label="Address" fullWidth />} />
              </FormSection>

              <FormSection title="Service scope">
                <Controller
                  name="servesAllProperties"
                  control={control}
                  render={({ field }) => (
                    <FormControlLabel
                      control={<Checkbox checked={field.value} onChange={(e) => field.onChange(e.target.checked)} />}
                      label="Serves all properties"
                    />
                  )}
                />
                {!servesAllProperties && (
                  <Controller
                    name="propertiesServed"
                    control={control}
                    render={({ field }) => (
                      <TextField
                        {...field}
                        select
                        label="Properties served"
                        fullWidth
                        slotProps={{
                          select: {
                            multiple: true,
                            renderValue: (selected) => `${(selected as string[]).length} selected`,
                          },
                        }}
                      >
                        {properties.map((p) => (
                          <MenuItem key={p.id} value={p.id}>
                            {p.name}
                          </MenuItem>
                        ))}
                      </TextField>
                    )}
                  />
                )}
              </FormSection>

              <FormSection title="Compliance">
                <Stack direction="row" spacing={2} sx={{ alignItems: 'flex-start' }}>
                  <Controller
                    name="insuranceExpiry"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Insurance expiry" type="date" fullWidth slotProps={{ inputLabel: { shrink: true } }} />}
                  />
                  <Controller
                    name="coiAttachmentId"
                    control={control}
                    render={({ field }) => (
                      <FileUpload
                        label="Certificate of insurance"
                        value={field.value ?? ''}
                        filename={coiName}
                        onChange={(id, name) => {
                          field.onChange(id);
                          setCoiName(name);
                        }}
                      />
                    )}
                  />
                </Stack>
                <Stack direction="row" spacing={2}>
                  <Controller name="licenseNumber" control={control} render={({ field }) => <TextField {...field} label="License number" fullWidth />} />
                  <Controller
                    name="licenseExpiry"
                    control={control}
                    render={({ field }) => <TextField {...field} label="License expiry" type="date" fullWidth slotProps={{ inputLabel: { shrink: true } }} />}
                  />
                </Stack>
                <Controller
                  name="taxDocAttachmentId"
                  control={control}
                  render={({ field }) => (
                    <FileUpload
                      label="Tax document (W-9, etc.)"
                      value={field.value ?? ''}
                      filename={taxDocName}
                      onChange={(id, name) => {
                        field.onChange(id);
                        setTaxDocName(name);
                      }}
                    />
                  )}
                />
              </FormSection>

              <FormSection title="Billing">
                <Stack direction="row" spacing={2}>
                  <Controller
                    name="rateType"
                    control={control}
                    render={({ field }) => (
                      <TextField {...field} select label="Rate type" fullWidth>
                        <MenuItem value="">
                          <em>Not specified</em>
                        </MenuItem>
                        {VENDOR_RATE_TYPES.map((t) => (
                          <MenuItem key={t} value={t}>
                            {VENDOR_RATE_TYPE_LABELS[t]}
                          </MenuItem>
                        ))}
                      </TextField>
                    )}
                  />
                  <Controller
                    name="rateAmount"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Rate amount" type="number" fullWidth slotProps={{ input: { startAdornment: '$' } }} />}
                  />
                </Stack>
                <Controller
                  name="paymentTerms"
                  control={control}
                  render={({ field }) => (
                    <TextField {...field} select label="Payment terms" fullWidth>
                      <MenuItem value="">
                        <em>Not specified</em>
                      </MenuItem>
                      {VENDOR_PAYMENT_TERMS.map((t) => (
                        <MenuItem key={t} value={t}>
                          {VENDOR_PAYMENT_TERMS_LABELS[t]}
                        </MenuItem>
                      ))}
                    </TextField>
                  )}
                />
              </FormSection>

              {isEditMode && (
                <FormSection title="Status">
                  <Controller
                    name="active"
                    control={control}
                    render={({ field }) => (
                      <FormControlLabel
                        control={<Checkbox checked={field.value} onChange={(e) => field.onChange(e.target.checked)} />}
                        label="Active"
                      />
                    )}
                  />
                </FormSection>
              )}

              <FormSection title="Internal notes">
                <Controller name="internalNotes" control={control} render={({ field }) => <TextField {...field} fullWidth multiline minRows={3} />} />
              </FormSection>
            </Stack>
          </Box>
        )}
      </DialogContent>

      {!isWaitingForRecord && (
        <>
          <Divider />
          <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'flex-end', px: 3, py: 2 }}>
            <Stack direction="row" spacing={1.5}>
              <Button variant="outlined" onClick={onClose} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
                Cancel
              </Button>
              <Button
                type="submit"
                form="vendor-form"
                variant="contained"
                disabled={isPending}
                startIcon={isPending ? <CircularProgress size={15} color="inherit" /> : undefined}
              >
                {isPending ? 'Saving…' : isEditMode ? 'Save changes' : 'Add Vendor'}
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
