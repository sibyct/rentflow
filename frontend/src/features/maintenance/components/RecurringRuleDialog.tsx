import { useEffect, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import type { PropertyRow } from '@/features/properties/mock/propertyRows';
import { useUnits } from '@/features/units/hooks/useUnitsQueries';
import { toFormValues } from '../api/recurringRulesApi';
import { useCreateRecurringRule, useUpdateRecurringRule } from '../hooks/useRecurringRulesQueries';
import { recurringRuleDefaultValues, recurringRuleSchema, type RecurringRuleFormValues } from '../schemas/recurringRuleSchema';
import { RECURRING_RULE_FREQUENCY_UNIT_LABELS, RECURRING_RULE_FREQUENCY_UNITS, WORK_ORDER_CATEGORIES, WORK_ORDER_CATEGORY_LABELS, type RecurringRuleRow } from '../types';

interface RecurringRuleDialogProps {
  open: boolean;
  onClose: () => void;
  properties: PropertyRow[];
  /** Present = edit an existing rule; absent = create a new one. */
  rule?: RecurringRuleRow;
  onSaved: () => void;
}

export function RecurringRuleDialog({ open, onClose, properties, rule, onSaved }: RecurringRuleDialogProps) {
  const isEditMode = Boolean(rule);
  const [propertyId, setPropertyId] = useState(rule?.propertyId ?? '');
  const { data: units } = useUnits(propertyId || undefined);
  const createRule = useCreateRecurringRule(propertyId);
  const updateRule = useUpdateRecurringRule();
  const [submitError, setSubmitError] = useState<string | null>(null);

  const {
    control,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<RecurringRuleFormValues>({
    resolver: zodResolver(recurringRuleSchema),
    defaultValues: recurringRuleDefaultValues(),
  });

  useEffect(() => {
    if (!open) return;
    reset(rule ? toFormValues(rule) : recurringRuleDefaultValues());
    setPropertyId(rule?.propertyId ?? '');
    setSubmitError(null);
  }, [open, rule, reset]);

  function onSubmit(values: RecurringRuleFormValues) {
    setSubmitError(null);
    const onError = (err: unknown) => setSubmitError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.');

    if (isEditMode && rule) {
      updateRule.mutate({ id: rule.id, values }, { onSuccess: onSaved, onError });
    } else {
      createRule.mutate(values, { onSuccess: onSaved, onError });
    }
  }

  const isSaving = createRule.isPending || updateRule.isPending;
  const needsPropertyPicker = !isEditMode && !propertyId;

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>{isEditMode ? 'Edit Recurring Rule' : 'New Recurring Rule'}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          {submitError && <Alert severity="error">{submitError}</Alert>}

          {needsPropertyPicker ? (
            <TextField select label="Property" value={propertyId} onChange={(e) => setPropertyId(e.target.value)} fullWidth autoFocus>
              {properties.map((p) => (
                <MenuItem key={p.id} value={p.id}>
                  {p.name}
                </MenuItem>
              ))}
            </TextField>
          ) : (
            <Box component="form" id="recurring-rule-form" onSubmit={handleSubmit(onSubmit)}>
              <Stack spacing={2}>
                {!isEditMode && (
                  <Controller
                    name="unitId"
                    control={control}
                    render={({ field }) => (
                      <TextField {...field} select label="Unit" fullWidth>
                        <MenuItem value="">
                          <em>Property-wide (no specific unit)</em>
                        </MenuItem>
                        {(units ?? []).map((u) => (
                          <MenuItem key={u.id} value={u.id}>
                            {u.unitName}
                          </MenuItem>
                        ))}
                      </TextField>
                    )}
                  />
                )}
                <Controller
                  name="title"
                  control={control}
                  render={({ field }) => <TextField {...field} label="Title" required fullWidth placeholder="e.g. HVAC filter change" error={Boolean(errors.title)} helperText={errors.title?.message} />}
                />
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
                  render={({ field }) => <TextField {...field} label="Description" fullWidth multiline minRows={2} />}
                />
                <Stack direction="row" spacing={2}>
                  <Controller
                    name="frequencyInterval"
                    control={control}
                    render={({ field }) => <TextField {...field} label="Every" type="number" fullWidth error={Boolean(errors.frequencyInterval)} helperText={errors.frequencyInterval?.message} />}
                  />
                  <Controller
                    name="frequencyUnit"
                    control={control}
                    render={({ field }) => (
                      <TextField {...field} select label="Frequency" fullWidth>
                        {RECURRING_RULE_FREQUENCY_UNITS.map((u) => (
                          <MenuItem key={u} value={u}>
                            {RECURRING_RULE_FREQUENCY_UNIT_LABELS[u]}
                          </MenuItem>
                        ))}
                      </TextField>
                    )}
                  />
                </Stack>
                <Controller
                  name="nextDueDate"
                  control={control}
                  render={({ field }) => (
                    <TextField {...field} label="Next due date" type="date" required fullWidth slotProps={{ inputLabel: { shrink: true } }} error={Boolean(errors.nextDueDate)} helperText={errors.nextDueDate?.message} />
                  )}
                />
              </Stack>
            </Box>
          )}
        </Stack>
      </DialogContent>
      <DialogActions sx={{ p: 2.5, pt: 0 }}>
        <Button variant="text" onClick={onClose} sx={{ color: tokens.slate[600] }}>
          Cancel
        </Button>
        {!needsPropertyPicker && (
          <Button type="submit" form="recurring-rule-form" variant="contained" disabled={isSaving}>
            {isSaving ? 'Saving…' : isEditMode ? 'Save changes' : 'Create Rule'}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
}
