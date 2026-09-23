import { useEffect, useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { Controller, useForm, useWatch } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import Divider from '@mui/material/Divider';
import Drawer from '@mui/material/Drawer';
import FormControlLabel from '@mui/material/FormControlLabel';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Switch from '@mui/material/Switch';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import CloseOutlined from '@mui/icons-material/CloseOutlined';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import type { PropertyRow } from '@/features/properties/mock/propertyRows';
import { useUnitsPortfolio } from '@/features/units/hooks/useUnitsQueries';
import { useVendors } from '@/features/vendors/hooks/useVendorsQueries';
import { FileUpload, FormSection, StatusChip } from '@/shared/components';
import { centsToDollarsInput, formatDate, formatMoney } from '@/shared/lib/format';
import {
  useCreateExpense,
  useExpense,
  usePayments,
  useRecordPayment,
  useTransactionAudit,
  useUpdateExpense,
  useVoidPayment,
  useVoidTransaction,
} from '../hooks/useAccountingQueries';
import { expenseDefaultValues, expenseSchema, type ExpenseFormValues } from '../schemas/expenseSchema';
import {
  EXPENSE_CATEGORIES,
  EXPENSE_CATEGORY_LABELS,
  EXPENSE_STATUS_LABELS,
  EXPENSE_STATUS_TONE,
  PAYMENT_METHODS,
  RECURRENCE_FREQUENCIES,
  RECURRENCE_LABELS,
  type ExpenseStatus,
  type LedgerRow,
} from '../types';
import { RecordPaymentDialog } from './RecordPaymentDialog';

function toFormValues(row: LedgerRow): ExpenseFormValues {
  return {
    propertyId: row.propertyId,
    unitId: row.unitId,
    category: row.category as ExpenseFormValues['category'],
    vendorId: row.vendorId,
    // A linked vendor's company name comes back in vendorName too; only keep
    // it as the freeform fallback when there is no vendor record.
    vendorName: row.vendorId ? '' : row.vendorName,
    amount: centsToDollarsInput(row.amountCents),
    incurredOn: row.incurredOn,
    dueOn: row.dueOn,
    description: row.description,
    taxDeductible: row.taxDeductible,
    isRecurring: row.isRecurring,
    recurrenceFrequency: row.recurrenceFrequency,
    paidOn: '',
    paymentMethod: '',
    attachmentId: row.attachmentId,
  };
}

interface ExpenseDrawerProps {
  open: boolean;
  onClose: () => void;
  properties: PropertyRow[];
  /** Present = edit/inspect an existing expense; absent = add a new one. */
  expenseId?: string;
  onSaved: (message: string) => void;
}

/**
 * Add / edit an expense in a right-hand drawer (the expenses list stays
 * visible behind it, like the work order drawer). Edit mode also shows
 * the expense's payments and change log. An expense created from a
 * completed work order is read-only here — its amount mirrors the work
 * order's actual cost, so it is edited there.
 */
export function ExpenseDrawer({ open, onClose, properties, expenseId, onSaved }: ExpenseDrawerProps) {
  const isEdit = Boolean(expenseId);
  const { data: existing } = useExpense(expenseId);
  const { data: payments } = usePayments(expenseId);
  const { data: audit } = useTransactionAudit(expenseId);
  const create = useCreateExpense();
  const update = useUpdateExpense(expenseId ?? '');
  const recordPayment = useRecordPayment();
  const voidPayment = useVoidPayment();
  const voidTransaction = useVoidTransaction();

  const [submitError, setSubmitError] = useState<string | null>(null);
  const [payOpen, setPayOpen] = useState(false);
  const [payError, setPayError] = useState<string | null>(null);
  const [confirmVoid, setConfirmVoid] = useState(false);
  // Name of a receipt uploaded in this session (an existing one shows a generic label).
  const [receiptName, setReceiptName] = useState('');

  const {
    control,
    handleSubmit,
    reset,
    setValue,
    formState: { errors },
  } = useForm<ExpenseFormValues>({ resolver: zodResolver(expenseSchema), defaultValues: expenseDefaultValues() });

  const propertyId = useWatch({ control, name: 'propertyId' });
  const vendorId = useWatch({ control, name: 'vendorId' });
  const isRecurring = useWatch({ control, name: 'isRecurring' });
  const paidOn = useWatch({ control, name: 'paidOn' });

  const { data: unitsData } = useUnitsPortfolio({ propertyId: propertyId || undefined, limit: 100, offset: 0 });
  const units = propertyId ? (unitsData?.units ?? []) : [];
  const { data: vendorsData } = useVendors({ limit: 100, offset: 0 });
  const vendors = vendorsData?.vendors ?? [];

  useEffect(() => {
    if (!open) return;
    setSubmitError(null);
    setReceiptName('');
    if (isEdit) {
      if (existing) reset(toFormValues(existing));
    } else {
      reset(expenseDefaultValues(properties.length === 1 ? (properties[0]?.id ?? '') : ''));
    }
  }, [open, isEdit, existing, properties, reset]);

  const readOnly = Boolean(existing?.readOnly);
  const livePayments = (payments ?? []).filter((p) => !p.voided);
  const status = existing?.status as ExpenseStatus | undefined;
  const saving = create.isPending || update.isPending;

  function onSubmit(values: ExpenseFormValues) {
    setSubmitError(null);
    const onError = (err: unknown) =>
      setSubmitError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.');
    if (isEdit) {
      update.mutate(values, { onSuccess: () => onSaved('Expense updated'), onError });
    } else {
      create.mutate(values, { onSuccess: () => onSaved('Expense added'), onError });
    }
  }

  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={saving ? undefined : onClose}
      slotProps={{ paper: { sx: { width: { xs: '100%', sm: 560 }, display: 'flex', flexDirection: 'column' } } }}
    >
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', p: 2.5, pb: 1.5 }}>
        <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center' }}>
          <Typography variant="h5">{isEdit ? 'Expense' : 'Add expense'}</Typography>
          {status && <StatusChip label={EXPENSE_STATUS_LABELS[status]} tone={EXPENSE_STATUS_TONE[status]} />}
        </Stack>
        <IconButton onClick={onClose} aria-label="Close">
          <CloseOutlined />
        </IconButton>
      </Stack>
      <Divider />

      <Box component="form" id="expense-form" onSubmit={handleSubmit(onSubmit)} noValidate sx={{ flex: 1, overflowY: 'auto', p: 2.5 }}>
        <Stack spacing={3}>
          {submitError && <Alert severity="error">{submitError}</Alert>}
          {readOnly && existing && (
            <Alert severity="info">
              Created from work order{' '}
              <Link component={RouterLink} to="/maintenance">
                {existing.workOrderTitle || 'work order'}
              </Link>
              . Its amount follows the work order&apos;s actual cost — change it there.
            </Alert>
          )}

          <FormSection title="Where">
            <Controller
              name="propertyId"
              control={control}
              render={({ field }) => (
                <TextField
                  {...field}
                  select
                  required
                  label="Property"
                  disabled={readOnly}
                  error={Boolean(errors.propertyId)}
                  helperText={errors.propertyId?.message}
                  onChange={(e) => {
                    field.onChange(e);
                    setValue('unitId', '');
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
            <Controller
              name="unitId"
              control={control}
              render={({ field }) => (
                <TextField
                  {...field}
                  select
                  label="Unit (optional)"
                  disabled={readOnly || !propertyId}
                  slotProps={{ inputLabel: { shrink: true }, select: { displayEmpty: true } }}
                >
                  <MenuItem value="">Whole property</MenuItem>
                  {units.map((u) => (
                    <MenuItem key={u.id} value={u.id}>
                      {u.unitName}
                    </MenuItem>
                  ))}
                </TextField>
              )}
            />
          </FormSection>

          <FormSection title="Expense">
            <Controller
              name="category"
              control={control}
              render={({ field }) => (
                <TextField {...field} value={field.value ?? ''} select required label="Category" disabled={readOnly} error={Boolean(errors.category)} helperText={errors.category?.message}>
                  {EXPENSE_CATEGORIES.map((c) => (
                    <MenuItem key={c} value={c}>
                      {EXPENSE_CATEGORY_LABELS[c]}
                    </MenuItem>
                  ))}
                </TextField>
              )}
            />
            <Stack direction="row" spacing={2}>
              <Controller
                name="amount"
                control={control}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Amount"
                    type="number"
                    required
                    disabled={readOnly}
                    sx={{ flex: 1 }}
                    error={Boolean(errors.amount)}
                    helperText={errors.amount?.message}
                    slotProps={{ input: { startAdornment: '$' }, htmlInput: { step: '0.01', min: 0 } }}
                  />
                )}
              />
              <Controller
                name="incurredOn"
                control={control}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Date incurred"
                    type="date"
                    required
                    disabled={readOnly}
                    sx={{ flex: 1 }}
                    error={Boolean(errors.incurredOn)}
                    helperText={errors.incurredOn?.message}
                    slotProps={{ inputLabel: { shrink: true } }}
                  />
                )}
              />
            </Stack>
            <Controller
              name="dueOn"
              control={control}
              render={({ field }) => (
                <TextField
                  {...field}
                  label="Due date (optional)"
                  type="date"
                  disabled={readOnly}
                  error={Boolean(errors.dueOn)}
                  helperText={errors.dueOn?.message ?? 'Unpaid expenses turn Overdue after this date.'}
                  slotProps={{ inputLabel: { shrink: true } }}
                />
              )}
            />
            <Controller
              name="description"
              control={control}
              render={({ field }) => <TextField {...field} label="Description" multiline minRows={2} disabled={readOnly} />}
            />
          </FormSection>

          <FormSection title="Vendor">
            <Controller
              name="vendorId"
              control={control}
              render={({ field }) => (
                <TextField {...field} select label="Vendor (optional)" disabled={readOnly} slotProps={{ inputLabel: { shrink: true }, select: { displayEmpty: true } }}>
                  <MenuItem value="">No vendor record</MenuItem>
                  {vendors.map((v) => (
                    <MenuItem key={v.id} value={v.id}>
                      {v.companyName}
                    </MenuItem>
                  ))}
                </TextField>
              )}
            />
            {!vendorId && (
              <Controller
                name="vendorName"
                control={control}
                render={({ field }) => <TextField {...field} label="Paid to" placeholder="Utility company, county, …" disabled={readOnly} />}
              />
            )}
          </FormSection>

          <FormSection title="Options">
            <Controller
              name="taxDeductible"
              control={control}
              render={({ field }) => (
                <FormControlLabel control={<Switch checked={field.value} onChange={field.onChange} disabled={readOnly} />} label="Tax-deductible" />
              )}
            />
            <Controller
              name="isRecurring"
              control={control}
              render={({ field }) => (
                <FormControlLabel control={<Switch checked={field.value} onChange={field.onChange} disabled={readOnly} />} label="Recurring expense" />
              )}
            />
            {isRecurring && (
              <Controller
                name="recurrenceFrequency"
                control={control}
                render={({ field }) => (
                  <TextField {...field} select label="Repeats" disabled={readOnly} error={Boolean(errors.recurrenceFrequency)} helperText={errors.recurrenceFrequency?.message}>
                    {RECURRENCE_FREQUENCIES.map((f) => (
                      <MenuItem key={f} value={f}>
                        {RECURRENCE_LABELS[f]}
                      </MenuItem>
                    ))}
                  </TextField>
                )}
              />
            )}
          </FormSection>

          <FormSection title="Receipt">
            <Controller
              name="attachmentId"
              control={control}
              render={({ field }) => (
                <FileUpload
                  label="Receipt or invoice (photo or PDF)"
                  value={field.value}
                  filename={receiptName}
                  disabled={readOnly}
                  onChange={(id, name) => {
                    field.onChange(id);
                    setReceiptName(name);
                  }}
                />
              )}
            />
          </FormSection>

          {!isEdit && (
            <FormSection title="Payment (optional)">
              <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>
                Already paid? Enter the date and it is recorded as paid in full.
              </Typography>
              <Stack direction="row" spacing={2}>
                <Controller
                  name="paidOn"
                  control={control}
                  render={({ field }) => (
                    <TextField {...field} label="Date paid" type="date" sx={{ flex: 1 }} slotProps={{ inputLabel: { shrink: true } }} />
                  )}
                />
                <Controller
                  name="paymentMethod"
                  control={control}
                  render={({ field }) => (
                    <TextField {...field} select label="Method" disabled={!paidOn} sx={{ flex: 1 }} slotProps={{ inputLabel: { shrink: true }, select: { displayEmpty: true } }}>
                      <MenuItem value="">—</MenuItem>
                      {PAYMENT_METHODS.map((m) => (
                        <MenuItem key={m} value={m}>
                          {m}
                        </MenuItem>
                      ))}
                    </TextField>
                  )}
                />
              </Stack>
            </FormSection>
          )}

          {isEdit && existing && (
            <FormSection title="Payments">
              {livePayments.length === 0 && <Typography sx={{ fontSize: 13, color: tokens.slate[500] }}>No payments recorded.</Typography>}
              {(payments ?? []).map((p) => (
                <Stack key={p.id} direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', opacity: p.voided ? 0.5 : 1 }}>
                  <Box>
                    <Typography sx={{ fontFamily: tokens.fontMono, fontSize: 13, textDecoration: p.voided ? 'line-through' : 'none' }}>
                      {formatMoney(p.amountCents)}
                    </Typography>
                    <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>
                      {formatDate(p.paidOn)}
                      {p.method ? ` · ${p.method}` : ''}
                      {p.reference ? ` · ${p.reference}` : ''}
                      {p.voided ? ' · voided' : ''}
                    </Typography>
                  </Box>
                  {!p.voided && (
                    <Button size="small" color="error" variant="text" disabled={voidPayment.isPending} onClick={() => voidPayment.mutate(p.id)}>
                      Void
                    </Button>
                  )}
                </Stack>
              ))}
              <Stack direction="row" spacing={1}>
                <Button
                  size="small"
                  variant="outlined"
                  disabled={existing.outstandingCents <= 0}
                  onClick={() => {
                    setPayError(null);
                    setPayOpen(true);
                  }}
                  sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}
                >
                  Record payment
                </Button>
                {livePayments.length === 0 && (
                  <Button size="small" color="error" variant="text" onClick={() => setConfirmVoid(true)}>
                    Void expense
                  </Button>
                )}
              </Stack>
            </FormSection>
          )}

          {isEdit && audit && audit.length > 0 && (
            <FormSection title="Change log">
              {audit.map((e) => (
                <Box key={e.id}>
                  <Typography sx={{ fontSize: 12.5, fontWeight: 600, color: tokens.slate[700] }}>
                    {e.action.replace(/_/g, ' ')} · {formatDate(e.createdAt)}
                  </Typography>
                  {Object.entries(e.changes).map(([field, c]) => (
                    <Typography key={field} sx={{ fontSize: 12, color: tokens.slate[500] }}>
                      {field.replace(/_/g, ' ')}: {c.old == null ? '—' : String(c.old)} → {c.new == null ? '—' : String(c.new)}
                    </Typography>
                  ))}
                </Box>
              ))}
            </FormSection>
          )}
        </Stack>
      </Box>

      <Divider />
      <Stack direction="row" spacing={1.5} sx={{ p: 2, justifyContent: 'flex-end' }}>
        <Button variant="text" onClick={onClose} sx={{ color: tokens.slate[600] }}>
          {readOnly ? 'Close' : 'Cancel'}
        </Button>
        {!readOnly && (
          <Button type="submit" form="expense-form" variant="contained" disabled={saving} startIcon={saving ? <CircularProgress size={15} color="inherit" /> : undefined}>
            {saving ? 'Saving…' : isEdit ? 'Save changes' : 'Add expense'}
          </Button>
        )}
      </Stack>

      <RecordPaymentDialog
        open={payOpen}
        title="Record payment"
        subtitle={existing ? `${existing.propertyName} — ${existing.description || EXPENSE_CATEGORY_LABELS[existing.category as keyof typeof EXPENSE_CATEGORY_LABELS] || 'Expense'}` : undefined}
        outstandingCents={existing?.outstandingCents ?? 0}
        submitting={recordPayment.isPending}
        error={payError}
        onClose={() => setPayOpen(false)}
        onSubmit={(values) => {
          if (!expenseId) return;
          recordPayment.mutate(
            { transactionId: expenseId, values },
            {
              onSuccess: () => setPayOpen(false),
              onError: (err) => setPayError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.'),
            },
          );
        }}
      />

      <Dialog open={confirmVoid} onClose={() => setConfirmVoid(false)}>
        <DialogTitle>Void this expense?</DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>
            It disappears from the expenses list and totals. The change log keeps a record that it existed.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 0 }}>
          <Button variant="text" onClick={() => setConfirmVoid(false)} sx={{ color: tokens.slate[600] }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color="error"
            disabled={voidTransaction.isPending}
            onClick={() => {
              if (!expenseId) return;
              voidTransaction.mutate(expenseId, {
                onSuccess: () => {
                  setConfirmVoid(false);
                  onSaved('Expense voided');
                },
                onError: (err) => {
                  setConfirmVoid(false);
                  setSubmitError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.');
                },
              });
            }}
          >
            Void expense
          </Button>
        </DialogActions>
      </Dialog>
    </Drawer>
  );
}
