import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import MenuItem from '@mui/material/MenuItem';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import AutorenewOutlined from '@mui/icons-material/AutorenewOutlined';
import TuneOutlined from '@mui/icons-material/TuneOutlined';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import { useProperties } from '@/features/properties/hooks/usePropertiesQueries';
import { ErrorBoundary, ErrorMessage } from '@/shared/components';
import { currentMonthIso, dollarsToCents, formatMoney, formatMonth } from '@/shared/lib/format';
import type { RentRollSortKey } from '../api/accountingApi';
import { useGenerateRent, useRecordLeasePayment, useRentRoll } from '../hooks/useAccountingQueries';
import { PAYMENT_STATUSES, PAYMENT_STATUS_LABELS, type PaymentStatus, type RentRollRow } from '../types';
import { LateFeeSettingsDialog } from './LateFeeSettingsDialog';
import { RecordPaymentDialog } from './RecordPaymentDialog';
import { RentRollTable, type RentRollEmptyState } from './RentRollTable';

/** /accounting/rent-roll — every lease's rent for a month (or a range), with payments recorded right from the row. */
export function RentRollScreen() {
  const [searchParams] = useSearchParams();

  const [propertyFilter, setPropertyFilter] = useState('');
  // Seeded once from ?status=late — the accounting dashboard's Outstanding card links here.
  const [statusFilter, setStatusFilter] = useState<PaymentStatus | ''>(
    (searchParams.get('status') as PaymentStatus | null) ?? '',
  );
  const [from, setFrom] = useState(currentMonthIso());
  const [to, setTo] = useState(currentMonthIso());
  const [sortKey, setSortKey] = useState<RentRollSortKey>('property');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [payTarget, setPayTarget] = useState<RentRollRow | null>(null);
  const [payError, setPayError] = useState<string | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);

  const { data: propertiesData } = useProperties({ limit: 100, offset: 0 });
  const properties = propertiesData?.properties ?? [];

  const { data, isLoading, isError, error, refetch } = useRentRoll({
    propertyId: propertyFilter || undefined,
    status: statusFilter || undefined,
    from,
    to: to < from ? from : to,
    sort: sortKey,
    order: sortDir,
    limit: rowsPerPage,
    offset: page * rowsPerPage,
  });
  const generate = useGenerateRent();
  const recordPayment = useRecordLeasePayment();

  const rows = data?.rows ?? [];
  const total = data?.total ?? 0;
  const isFiltered = Boolean(propertyFilter || statusFilter);
  const emptyState: RentRollEmptyState =
    isLoading || total > 0 ? { kind: 'none' } : isFiltered ? { kind: 'filtered' } : { kind: 'first-time' };

  // Rent for a month is only generated up to the current one.
  const canGenerate = from <= currentMonthIso();

  function handleSort(key: RentRollSortKey) {
    if (sortKey === key) setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    else {
      setSortKey(key);
      setSortDir('asc');
    }
    setPage(0);
  }

  function resetView() {
    setPropertyFilter('');
    setStatusFilter('');
    setFrom(currentMonthIso());
    setTo(currentMonthIso());
    setPage(0);
  }

  return (
    <Box>
      <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center', flexWrap: 'wrap', mb: 2, rowGap: 1.25 }}>
        <TextField
          select
          size="small"
          slotProps={{ select: { displayEmpty: true } }}
          value={propertyFilter}
          onChange={(e) => {
            setPropertyFilter(e.target.value);
            setPage(0);
          }}
          sx={{ minWidth: 180 }}
        >
          <MenuItem value="">Property: All</MenuItem>
          {properties.map((p) => (
            <MenuItem key={p.id} value={p.id}>
              {p.name}
            </MenuItem>
          ))}
        </TextField>
        <TextField
          select
          size="small"
          slotProps={{ select: { displayEmpty: true } }}
          value={statusFilter}
          onChange={(e) => {
            setStatusFilter(e.target.value as PaymentStatus | '');
            setPage(0);
          }}
          sx={{ minWidth: 150 }}
        >
          <MenuItem value="">Status: All</MenuItem>
          {PAYMENT_STATUSES.map((s) => (
            <MenuItem key={s} value={s}>
              {PAYMENT_STATUS_LABELS[s]}
            </MenuItem>
          ))}
        </TextField>
        <TextField
          size="small"
          type="month"
          label="From"
          value={from}
          onChange={(e) => {
            if (!e.target.value) return;
            setFrom(e.target.value);
            if (to < e.target.value) setTo(e.target.value);
            setPage(0);
          }}
          slotProps={{ inputLabel: { shrink: true } }}
          sx={{ width: 160 }}
        />
        <TextField
          size="small"
          type="month"
          label="To"
          value={to}
          onChange={(e) => {
            if (!e.target.value) return;
            setTo(e.target.value);
            setPage(0);
          }}
          slotProps={{ inputLabel: { shrink: true }, htmlInput: { min: from } }}
          sx={{ width: 160 }}
        />

        <Box sx={{ flex: 1 }} />

        <Button
          size="small"
          variant="outlined"
          startIcon={<TuneOutlined />}
          onClick={() => setSettingsOpen(true)}
          sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}
        >
          Late fee rules
        </Button>
        <Button
          size="small"
          variant="contained"
          startIcon={<AutorenewOutlined />}
          disabled={!canGenerate || generate.isPending}
          onClick={() =>
            generate.mutate(`${from}-01`, {
              onSuccess: (res) =>
                setToast({
                  message:
                    res.rent_created + res.late_fee_created === 0
                      ? `${formatMonth(`${from}-01`)} is already up to date`
                      : `Generated ${res.rent_created} rent and ${res.late_fee_created} late-fee rows`,
                  severity: 'success',
                }),
              onError: (err) => setToast({ message: err instanceof ApiError ? err.message : 'Something went wrong.', severity: 'error' }),
            })
          }
        >
          Generate {formatMonth(`${from}-01`)}
        </Button>
      </Stack>

      {isError ? (
        <ErrorMessage error={error} onRetry={() => void refetch()} />
      ) : (
        <ErrorBoundary>
          <RentRollTable
            loading={isLoading}
            rows={rows}
            total={total}
            emptyState={emptyState}
            sortKey={sortKey}
            sortDir={sortDir}
            onSort={handleSort}
            page={page}
            rowsPerPage={rowsPerPage}
            onPageChange={setPage}
            onRowsPerPageChange={(n) => {
              setRowsPerPage(n);
              setPage(0);
            }}
            onResetView={resetView}
            onRecordPayment={(r) => {
              setPayError(null);
              setPayTarget(r);
            }}
          />
        </ErrorBoundary>
      )}

      <RecordPaymentDialog
        open={Boolean(payTarget)}
        title="Record payment"
        subtitle={payTarget ? `${payTarget.propertyName} · ${payTarget.unitName} — ${payTarget.tenantName}` : undefined}
        outstandingCents={payTarget?.outstandingCents ?? 0}
        submitting={recordPayment.isPending}
        error={payError}
        onClose={() => setPayTarget(null)}
        onSubmit={(values) => {
          if (!payTarget) return;
          recordPayment.mutate(
            { leaseId: payTarget.leaseId, values },
            {
              onSuccess: () => {
                setToast({ message: `Payment of ${formatMoney(dollarsToCents(values.amount) ?? 0)} recorded`, severity: 'success' });
                setPayTarget(null);
              },
              onError: (err) => setPayError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.'),
            },
          );
        }}
      />

      <LateFeeSettingsDialog
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        onSaved={() => {
          setSettingsOpen(false);
          setToast({ message: 'Late fee rules saved', severity: 'success' });
        }}
      />

      <Snackbar open={Boolean(toast)} autoHideDuration={3200} onClose={() => setToast(null)} anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}>
        {toast ? (
          <Alert onClose={() => setToast(null)} severity={toast.severity} variant="filled" sx={{ borderRadius: 999 }}>
            {toast.message}
          </Alert>
        ) : undefined}
      </Snackbar>
    </Box>
  );
}
