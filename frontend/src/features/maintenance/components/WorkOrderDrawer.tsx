import { useEffect, useState, type ReactNode } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import Divider from '@mui/material/Divider';
import Drawer from '@mui/material/Drawer';
import IconButton from '@mui/material/IconButton';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import Typography from '@mui/material/Typography';
import AddCircleOutlineOutlined from '@mui/icons-material/AddCircleOutlineOutlined';
import CheckCircleOutlined from '@mui/icons-material/CheckCircleOutlined';
import CloseOutlined from '@mui/icons-material/CloseOutlined';
import DeleteOutlineOutlined from '@mui/icons-material/DeleteOutlineOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';
import LockOutlined from '@mui/icons-material/LockOutlined';
import VisibilityOutlined from '@mui/icons-material/VisibilityOutlined';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import type { PropertyRow } from '@/features/properties/mock/propertyRows';
import { useUnits } from '@/features/units/hooks/useUnitsQueries';
import { VendorRatingInput, VendorSelector } from '@/features/vendors/components/VendorSelector';
import { relativeTime } from '@/shared/lib/relativeTime';
import { toFormValues } from '../api/workOrdersApi';
import {
  useAddWorkOrderNote,
  useCreateWorkOrder,
  useDeleteWorkOrder,
  useUpdateWorkOrder,
  useWorkOrder,
  useWorkOrderActivity,
} from '../hooks/useWorkOrdersQueries';
import { workOrderDefaultValues, workOrderSchema, type WorkOrderFormValues } from '../schemas/workOrderSchema';
import {
  EMERGENCY_SLA_HOURS,
  WORK_ORDER_CATEGORIES,
  WORK_ORDER_CATEGORY_LABELS,
  WORK_ORDER_PRIORITIES,
  WORK_ORDER_PRIORITY_LABELS,
  WORK_ORDER_STATUSES,
  WORK_ORDER_STATUS_LABELS,
} from '../types';

interface WorkOrderDrawerProps {
  open: boolean;
  onClose: () => void;
  /** Fixed context (opened from a property/unit page) — no picker shown. Omit both for the global Maintenance page's "New Work Order", and pass `properties` so a property can be picked. */
  propertyId?: string;
  unitId?: string;
  properties?: PropertyRow[];
  /** Present = edit an existing work order; absent = create a new one. */
  workOrderId?: string;
  onSaved: () => void;
  onDeleted: () => void;
}

/**
 * Single always-editable drawer for a work order — read, create, and
 * edit all happen here rather than a separate view-then-edit split
 * (unlike Units/Properties): the spec calls for one drawer with a
 * header, sections, and Save/Cancel/Delete/status-transition buttons
 * all in the same place, so there's nothing a separate read-only view
 * would add.
 */
export function WorkOrderDrawer({ open, onClose, propertyId, unitId, properties, workOrderId, onSaved, onDeleted }: WorkOrderDrawerProps) {
  const isEditMode = Boolean(workOrderId);
  const [pickedPropertyId, setPickedPropertyId] = useState('');
  const effectivePropertyId = propertyId ?? pickedPropertyId;
  const needsPropertyPicker = !propertyId && !pickedPropertyId && !isEditMode;

  const { data: existingWorkOrder, isLoading: isLoadingWorkOrder } = useWorkOrder(workOrderId);
  const { data: activity } = useWorkOrderActivity(workOrderId);
  const { data: unitsForProperty } = useUnits(effectivePropertyId || undefined);
  const createWorkOrder = useCreateWorkOrder(effectivePropertyId);
  const updateWorkOrder = useUpdateWorkOrder();
  const deleteWorkOrder = useDeleteWorkOrder();
  const addNote = useAddWorkOrderNote(workOrderId ?? '');

  const [submitError, setSubmitError] = useState<string | null>(null);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
  const [completePromptOpen, setCompletePromptOpen] = useState(false);
  const [noteDraft, setNoteDraft] = useState('');
  const [noteVisibility, setNoteVisibility] = useState<'internal' | 'tenant_visible'>('internal');

  const {
    control,
    handleSubmit,
    reset,
    watch,
    setValue,
    getValues,
    formState: { errors },
  } = useForm<WorkOrderFormValues>({
    resolver: zodResolver(workOrderSchema),
    defaultValues: workOrderDefaultValues(),
  });

  useEffect(() => {
    if (!open) return;
    if (isEditMode) {
      if (existingWorkOrder) reset(toFormValues(existingWorkOrder));
    } else {
      reset({ ...workOrderDefaultValues(), unitId: unitId ?? '' });
    }
    setPickedPropertyId('');
    setSubmitError(null);
    setCompletePromptOpen(false);
    setNoteDraft('');
  }, [open, isEditMode, existingWorkOrder, unitId, reset]);

  const priority = watch('priority');
  const status = watch('status');
  const category = watch('category');
  const vendorId = watch('vendorId');
  const isSaving = createWorkOrder.isPending || updateWorkOrder.isPending;
  const isWaitingForRecord = isEditMode && (isLoadingWorkOrder || !existingWorkOrder);
  const pickedProperty = properties?.find((p) => p.id === effectivePropertyId);
  const title = isEditMode ? 'Edit Work Order' : 'New Work Order';

  function persist(values: WorkOrderFormValues, onSuccessMessage?: () => void) {
    setSubmitError(null);
    const onError = (err: unknown) => setSubmitError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.');

    if (isEditMode && workOrderId) {
      updateWorkOrder.mutate({ id: workOrderId, values }, { onSuccess: () => (onSuccessMessage ? onSuccessMessage() : onSaved()), onError });
    } else {
      createWorkOrder.mutate(values, { onSuccess: () => (onSuccessMessage ? onSuccessMessage() : onSaved()), onError });
    }
  }

  function onSubmit(values: WorkOrderFormValues) {
    persist(values);
  }

  function handleStatusShortcut(next: WorkOrderFormValues['status']) {
    if (next === 'completed' && !getValues('actualCost') && !getValues('photoLink')) {
      setCompletePromptOpen(true);
      return;
    }
    setValue('status', next);
    persist({ ...getValues(), status: next });
  }

  function confirmCompleteWithCost() {
    setValue('status', 'completed');
    setCompletePromptOpen(false);
    persist({ ...getValues(), status: 'completed' });
  }

  function handleDelete() {
    if (!workOrderId) return;
    deleteWorkOrder.mutate(workOrderId, { onSuccess: () => { setConfirmDeleteOpen(false); onDeleted(); } });
  }

  function submitNote() {
    if (!noteDraft.trim()) return;
    addNote.mutate({ message: noteDraft.trim(), visibility: noteVisibility }, { onSuccess: () => setNoteDraft('') });
  }

  return (
    <>
      <Drawer anchor="right" open={open} onClose={onClose} slotProps={{ paper: { sx: { width: { xs: '100%', sm: 640 } } } }}>
        <Stack sx={{ height: '100%' }}>
          <Stack direction="row" sx={{ alignItems: 'flex-start', justifyContent: 'space-between', p: 2.5, borderBottom: `1px solid ${tokens.slate[100]}` }}>
            <Box>
              <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
                <Typography variant="h5" sx={{ fontSize: 17 }}>{title}</Typography>
                {existingWorkOrder && (
                  <>
                    <Chip label={WORK_ORDER_PRIORITY_LABELS[existingWorkOrder.priority]} size="small" color={existingWorkOrder.priority === 'emergency' ? 'error' : 'default'} sx={{ fontWeight: 600 }} />
                    <Chip label={WORK_ORDER_STATUS_LABELS[existingWorkOrder.status]} size="small" variant="outlined" sx={{ fontWeight: 600 }} />
                  </>
                )}
              </Stack>
              {existingWorkOrder && (
                <Typography sx={{ fontSize: 12.5, color: tokens.slate[500], mt: 0.5 }}>
                  {existingWorkOrder.propertyName}
                  {existingWorkOrder.unitName ? ` / ${existingWorkOrder.unitName}` : ' (property-wide)'} · Created {relativeTime(existingWorkOrder.createdAt)}
                </Typography>
              )}
            </Box>
            <IconButton onClick={onClose} aria-label="Close">
              <CloseOutlined fontSize="small" />
            </IconButton>
          </Stack>

          {isWaitingForRecord ? (
            <Box sx={{ p: 3 }}>
              <Typography sx={{ fontSize: 13.5, color: tokens.slate[500] }}>Loading work order…</Typography>
            </Box>
          ) : needsPropertyPicker ? (
            <Stack spacing={2} sx={{ p: 3 }}>
              <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>Which property is this for?</Typography>
              <TextField select label="Property" value={pickedPropertyId} onChange={(e) => setPickedPropertyId(e.target.value)} fullWidth autoFocus>
                {(properties ?? []).map((p) => (
                  <MenuItem key={p.id} value={p.id}>
                    {p.name}
                  </MenuItem>
                ))}
              </TextField>
            </Stack>
          ) : (
            <Box component="form" id="work-order-form" onSubmit={handleSubmit(onSubmit)} sx={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0 }}>
              <Stack spacing={3} sx={{ p: 2.5, overflowY: 'auto', flex: 1 }}>
                {submitError && <Alert severity="error">{submitError}</Alert>}

                {pickedProperty && !propertyId && (
                  <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>For {pickedProperty.name}</Typography>
                )}

                <FormSection title="Details">
                  <Controller
                    name="title"
                    control={control}
                    render={({ field }) => (
                      <TextField {...field} label="Title" required fullWidth error={Boolean(errors.title)} helperText={errors.title?.message} />
                    )}
                  />
                  {!unitId && (
                    <Controller
                      name="unitId"
                      control={control}
                      render={({ field }) => (
                        <TextField {...field} select label="Unit" fullWidth>
                          <MenuItem value="">
                            <em>Property-wide (no specific unit)</em>
                          </MenuItem>
                          {(unitsForProperty ?? []).map((u) => (
                            <MenuItem key={u.id} value={u.id}>
                              {u.unitName}
                            </MenuItem>
                          ))}
                        </TextField>
                      )}
                    />
                  )}
                  <Controller
                    name="category"
                    control={control}
                    render={({ field }) => (
                      <TextField {...field} value={field.value ?? ''} select label="Category" required fullWidth error={Boolean(errors.category)} helperText={errors.category?.message}>
                        {WORK_ORDER_CATEGORIES.map((c) => (
                          <MenuItem key={c} value={c}>
                            {WORK_ORDER_CATEGORY_LABELS[c]}
                          </MenuItem>
                        ))}
                      </TextField>
                    )}
                  />
                  <Controller
                    name="description"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Description" fullWidth multiline minRows={3} />}
                  />
                  <Controller
                    name="photoLink"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Photo / video link" fullWidth placeholder="Paste a link (no file upload yet)" />}
                  />
                </FormSection>

                <FormSection title="Reported by">
                  <Stack direction="row" spacing={2}>
                    <Controller name="reportedBy" control={control} render={({ field }) => <TextField {...field} label="Reported by" fullWidth />} />
                    <Controller name="reportedByContact" control={control} render={({ field }) => <TextField {...field} label="Contact" fullWidth />} />
                  </Stack>
                  <Controller
                    name="accessInstructions"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Access instructions" fullWidth multiline minRows={2} placeholder="Entry permission, pets, preferred hours…" />}
                  />
                </FormSection>

                <FormSection title="Priority & status">
                  <Box>
                    <Typography sx={{ fontSize: 12, fontWeight: 600, color: tokens.slate[600], mb: 0.75 }}>Priority</Typography>
                    <Controller
                      name="priority"
                      control={control}
                      render={({ field }) => (
                        <ToggleButtonGroup exclusive value={field.value} onChange={(_e, v) => v && field.onChange(v)} sx={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 1, width: '100%' }}>
                          {WORK_ORDER_PRIORITIES.map((p) => (
                            <ToggleButton
                              key={p}
                              value={p}
                              sx={{
                                border: `1.5px solid ${tokens.slate[200]} !important`,
                                borderRadius: `${tokens.radiusControl}px !important`,
                                textTransform: 'none',
                                fontWeight: 600,
                                fontSize: 12.5,
                                py: 1,
                                '&.Mui-selected': p === 'emergency'
                                  ? { bgcolor: '#FDEDED', color: tokens.error, borderColor: `${tokens.error} !important` }
                                  : { bgcolor: tokens.azure[50], color: tokens.azure[700], borderColor: `${tokens.azure[500]} !important` },
                              }}
                            >
                              {WORK_ORDER_PRIORITY_LABELS[p]}
                            </ToggleButton>
                          ))}
                        </ToggleButtonGroup>
                      )}
                    />
                  </Box>
                  <Stack direction="row" spacing={2}>
                    <Controller
                      name="status"
                      control={control}
                      render={({ field }) => (
                        <TextField {...field} select label="Status" fullWidth>
                          {WORK_ORDER_STATUSES.map((s) => (
                            <MenuItem key={s} value={s}>
                              {WORK_ORDER_STATUS_LABELS[s]}
                            </MenuItem>
                          ))}
                        </TextField>
                      )}
                    />
                    <Controller
                      name="dueDate"
                      control={control}
                      render={({ field }) => (
                        <TextField
                          {...field}
                          label="Due date"
                          type="date"
                          required={priority === 'emergency'}
                          fullWidth
                          slotProps={{ inputLabel: { shrink: true } }}
                          error={Boolean(errors.dueDate)}
                          helperText={errors.dueDate?.message ?? (priority === 'emergency' ? `Within ${EMERGENCY_SLA_HOURS}h` : undefined)}
                        />
                      )}
                    />
                  </Stack>
                </FormSection>

                <FormSection title="Assignment">
                  <Controller
                    name="vendorId"
                    control={control}
                    render={({ field }) => (
                      <VendorSelector
                        category={category}
                        value={field.value ?? ''}
                        onChange={(vendor) => {
                          field.onChange(vendor?.id ?? '');
                          if (vendor) {
                            setValue('assignedTo', vendor.companyName);
                            setValue('assignedToContact', vendor.phone);
                          }
                        }}
                      />
                    )}
                  />
                  <Stack direction="row" spacing={2}>
                    <Controller
                      name="assignedTo"
                      control={control}
                      render={({ field }) => <TextField {...field} label="Assigned to" fullWidth placeholder="Staff or vendor name" helperText={vendorId ? 'Filled from the selected vendor' : 'No vendor selected — freeform staff assignment'} />}
                    />
                    <Controller name="assignedToContact" control={control} render={({ field }) => <TextField {...field} label="Contact" fullWidth />} />
                  </Stack>
                  {vendorId && (
                    <Controller
                      name="rating"
                      control={control}
                      render={({ field }) => (
                        <VendorRatingInput value={field.value ? Number(field.value) : null} onChange={(v) => field.onChange(v != null ? String(v) : '')} />
                      )}
                    />
                  )}
                  <Stack direction="row" spacing={2}>
                    <Controller name="scheduledStart" control={control} render={({ field }) => <TextField {...field} label="Scheduled start" type="datetime-local" fullWidth slotProps={{ inputLabel: { shrink: true } }} />} />
                    <Controller name="scheduledEnd" control={control} render={({ field }) => <TextField {...field} label="Scheduled end" type="datetime-local" fullWidth slotProps={{ inputLabel: { shrink: true } }} />} />
                  </Stack>
                </FormSection>

                <FormSection title="Cost tracking">
                  <Stack direction="row" spacing={2}>
                    <Controller name="estimatedCost" control={control} render={({ field }) => <TextField {...field} label="Estimated cost" type="number" fullWidth slotProps={{ input: { startAdornment: '$' } }} />} />
                    <Controller name="actualCost" control={control} render={({ field }) => <TextField {...field} label="Actual cost" type="number" fullWidth slotProps={{ input: { startAdornment: '$' } }} />} />
                  </Stack>
                  <Controller name="invoiceLink" control={control} render={({ field }) => <TextField {...field} label="Vendor invoice link" fullWidth placeholder="Paste a link (no file upload yet)" />} />
                </FormSection>

                <FormSection title="Internal notes">
                  <Typography sx={{ fontSize: 11.5, color: tokens.slate[400], display: 'flex', alignItems: 'center', gap: 0.5, mb: -1 }}>
                    <LockOutlined sx={{ fontSize: 13 }} /> Not visible to a tenant
                  </Typography>
                  <Controller name="internalNotes" control={control} render={({ field }) => <TextField {...field} fullWidth multiline minRows={2} />} />
                </FormSection>

                {isEditMode && (
                  <FormSection title="Activity">
                    <Stack spacing={1.5}>
                      {(activity ?? []).length === 0 && <Typography sx={{ fontSize: 13, color: tokens.slate[400] }}>No activity yet.</Typography>}
                      {(activity ?? []).map((a) => (
                        <Stack key={a.id} direction="row" spacing={1} sx={{ alignItems: 'flex-start' }}>
                          {a.kind === 'status_change' ? (
                            <EditOutlined sx={{ fontSize: 16, color: tokens.azure[600], mt: 0.25 }} />
                          ) : a.visibility === 'internal' ? (
                            <LockOutlined sx={{ fontSize: 16, color: tokens.slate[400], mt: 0.25 }} />
                          ) : (
                            <VisibilityOutlined sx={{ fontSize: 16, color: tokens.slate[400], mt: 0.25 }} />
                          )}
                          <Box sx={{ flex: 1 }}>
                            <Typography sx={{ fontSize: 13, color: tokens.slate[700] }}>{a.message}</Typography>
                            <Typography sx={{ fontSize: 11.5, color: tokens.slate[400] }}>{relativeTime(a.createdAt)}</Typography>
                          </Box>
                        </Stack>
                      ))}
                    </Stack>

                    <Stack spacing={1} sx={{ mt: 1.5 }}>
                      <TextField
                        value={noteDraft}
                        onChange={(e) => setNoteDraft(e.target.value)}
                        placeholder="Add an update or internal note…"
                        fullWidth
                        multiline
                        minRows={2}
                        size="small"
                      />
                      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
                        <ToggleButtonGroup
                          exclusive
                          size="small"
                          value={noteVisibility}
                          onChange={(_e, v) => v && setNoteVisibility(v)}
                        >
                          <ToggleButton value="internal" sx={{ textTransform: 'none', fontSize: 12 }}>
                            <LockOutlined sx={{ fontSize: 14, mr: 0.5 }} /> Internal
                          </ToggleButton>
                          <ToggleButton value="tenant_visible" sx={{ textTransform: 'none', fontSize: 12 }}>
                            <VisibilityOutlined sx={{ fontSize: 14, mr: 0.5 }} /> Tenant-visible
                          </ToggleButton>
                        </ToggleButtonGroup>
                        <Button size="small" startIcon={<AddCircleOutlineOutlined />} onClick={submitNote} disabled={!noteDraft.trim() || addNote.isPending}>
                          Add
                        </Button>
                      </Stack>
                    </Stack>
                  </FormSection>
                )}
              </Stack>

              <Divider />
              <Stack sx={{ p: 2.5 }} spacing={1.5}>
                {isEditMode && status !== 'completed' && status !== 'cancelled' && (
                  <Stack direction="row" spacing={1}>
                    {status !== 'in_progress' && (
                      <Button size="small" variant="outlined" onClick={() => handleStatusShortcut('in_progress')} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
                        Mark In Progress
                      </Button>
                    )}
                    <Button size="small" variant="outlined" color="success" startIcon={<CheckCircleOutlined fontSize="small" />} onClick={() => handleStatusShortcut('completed')}>
                      Mark Completed
                    </Button>
                  </Stack>
                )}
                <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
                  {isEditMode ? (
                    <Button variant="text" color="error" startIcon={<DeleteOutlineOutlined />} onClick={() => setConfirmDeleteOpen(true)}>
                      Delete
                    </Button>
                  ) : (
                    <span />
                  )}
                  <Stack direction="row" spacing={1.5}>
                    <Button variant="outlined" onClick={onClose} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
                      Cancel
                    </Button>
                    <Button
                      type="submit"
                      form="work-order-form"
                      variant="contained"
                      disabled={isSaving}
                      startIcon={isSaving ? <CircularProgress size={15} color="inherit" /> : undefined}
                    >
                      {isSaving ? 'Saving…' : 'Save'}
                    </Button>
                  </Stack>
                </Stack>
              </Stack>
            </Box>
          )}
        </Stack>
      </Drawer>

      <Dialog open={confirmDeleteOpen} onClose={() => setConfirmDeleteOpen(false)}>
        <DialogTitle>Delete this work order?</DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>This will be permanently removed. This can&apos;t be undone.</Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 0 }}>
          <Button variant="text" onClick={() => setConfirmDeleteOpen(false)} sx={{ color: tokens.slate[600] }}>
            Cancel
          </Button>
          <Button variant="contained" color="error" onClick={handleDelete} disabled={deleteWorkOrder.isPending}>
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={completePromptOpen} onClose={() => setCompletePromptOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>Mark as completed?</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>Add the actual cost and a completion photo link before closing this out — or skip and complete anyway.</Typography>
            <Controller name="actualCost" control={control} render={({ field }) => <TextField {...field} label="Actual cost" type="number" fullWidth size="small" slotProps={{ input: { startAdornment: '$' } }} />} />
            <Controller name="photoLink" control={control} render={({ field }) => <TextField {...field} label="Completion photo link" fullWidth size="small" />} />
            {vendorId && (
              <Controller
                name="rating"
                control={control}
                render={({ field }) => (
                  <VendorRatingInput value={field.value ? Number(field.value) : null} onChange={(v) => field.onChange(v != null ? String(v) : '')} />
                )}
              />
            )}
          </Stack>
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 0 }}>
          <Button variant="text" onClick={() => setCompletePromptOpen(false)} sx={{ color: tokens.slate[600] }}>
            Cancel
          </Button>
          <Button variant="contained" color="success" onClick={confirmCompleteWithCost}>
            Mark Completed
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}

function FormSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Stack spacing={1.5}>
      <Typography sx={{ fontSize: 11, fontWeight: 600, letterSpacing: '0.04em', color: tokens.slate[500], textTransform: 'uppercase' }}>{title}</Typography>
      {children}
    </Stack>
  );
}
