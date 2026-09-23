import { useState } from 'react';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TablePagination from '@mui/material/TablePagination';
import TableRow from '@mui/material/TableRow';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import AddOutlined from '@mui/icons-material/AddOutlined';
import DescriptionOutlined from '@mui/icons-material/DescriptionOutlined';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import { useProperties } from '@/features/properties/hooks/usePropertiesQueries';
import { EmptyState, ErrorBoundary, ErrorMessage, StatusChip, TableSkeletonRows, type TableSkeletonColumn } from '@/shared/components';
import { formatDate, formatMoney } from '@/shared/lib/format';
import { useCharges, useRecordPayment } from '../hooks/useAccountingQueries';
import {
  CHARGE_TYPE_LABELS,
  PAYMENT_STATUSES,
  PAYMENT_STATUS_LABELS,
  PAYMENT_STATUS_TONE,
  type LedgerRow,
  type PaymentStatus,
} from '../types';
import { ChargeDialog } from './ChargeDialog';
import { RecordPaymentDialog } from './RecordPaymentDialog';

const SHOW_MD = { xs: 'none', md: 'table-cell' } as const;
const SHOW_LG = { xs: 'none', lg: 'table-cell' } as const;

const SKELETON_COLUMNS: TableSkeletonColumn[] = [
  { shapes: [{ width: '70%' }] },
  { shapes: [{ width: '70%' }, { width: '40%' }] },
  { shapes: [{ width: '60%' }] },
  { shapes: [{ width: '70%' }] },
  { align: 'right', shapes: [{ width: 70 }] },
  { shapes: [{ variant: 'rounded', width: 64, height: 22 }] },
  { shapes: [{ variant: 'rounded', width: 96, height: 30 }] },
];

/** /accounting/charges — one-off tenant charges (utility rebills, damage, amenity fees). Unpaid charges also count toward the Rent Roll balance. */
export function ChargesScreen() {
  const [propertyFilter, setPropertyFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState<PaymentStatus | ''>('');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [newOpen, setNewOpen] = useState(false);
  const [payTarget, setPayTarget] = useState<LedgerRow | null>(null);
  const [payError, setPayError] = useState<string | null>(null);
  const [toast, setToast] = useState<string | null>(null);

  const { data: propertiesData } = useProperties({ limit: 100, offset: 0 });
  const properties = propertiesData?.properties ?? [];
  const { data, isLoading, isError, error, refetch } = useCharges({
    propertyId: propertyFilter || undefined,
    status: statusFilter || undefined,
    limit: rowsPerPage,
    offset: page * rowsPerPage,
  });
  const recordPayment = useRecordPayment();

  const rows = data?.rows ?? [];
  const total = data?.total ?? 0;
  const isFiltered = Boolean(propertyFilter || statusFilter);

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
        <Box sx={{ flex: 1 }} />
        <Button size="small" variant="contained" startIcon={<AddOutlined />} onClick={() => setNewOpen(true)}>
          New charge
        </Button>
      </Stack>

      {isError ? (
        <ErrorMessage error={error} onRetry={() => void refetch()} />
      ) : (
        <ErrorBoundary>
          <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
            <TableContainer>
              <Table size="small">
                {(isLoading || total > 0) && (
                  <TableHead>
                    <TableRow>
                      <TableCell>Due</TableCell>
                      <TableCell>Unit</TableCell>
                      <TableCell sx={{ display: SHOW_MD }}>Type</TableCell>
                      <TableCell sx={{ display: SHOW_LG }}>Description</TableCell>
                      <TableCell align="right">Amount</TableCell>
                      <TableCell>Status</TableCell>
                      <TableCell />
                    </TableRow>
                  </TableHead>
                )}
                <TableBody>
                  {isLoading && <TableSkeletonRows columns={SKELETON_COLUMNS} />}
                  {!isLoading && total === 0 && (
                    <TableRow>
                      <TableCell colSpan={7}>
                        <EmptyState
                          icon={DescriptionOutlined}
                          title={isFiltered ? 'No charges match your filters' : 'No one-off charges yet'}
                          description={
                            isFiltered
                              ? 'Try different filters.'
                              : 'Bill a tenant for a utility rebill, damage or amenity fee. It is added to their Rent Roll balance.'
                          }
                          action={
                            isFiltered ? undefined : (
                              <Button variant="contained" startIcon={<AddOutlined />} onClick={() => setNewOpen(true)}>
                                New charge
                              </Button>
                            )
                          }
                        />
                      </TableCell>
                    </TableRow>
                  )}
                  {!isLoading &&
                    rows.map((r) => {
                      const status = r.status as PaymentStatus;
                      const type = r.chargeType ? CHARGE_TYPE_LABELS[r.chargeType] : '—';
                      return (
                        <TableRow key={r.id} hover>
                          <TableCell>
                            <Typography sx={{ fontSize: 13 }}>{formatDate(r.dueOn)}</Typography>
                          </TableCell>
                          <TableCell>
                            <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{r.propertyName}</Typography>
                            <Typography sx={{ fontSize: 12.5, color: tokens.slate[600] }}>
                              {r.unitName}
                              {r.tenantName ? ` — ${r.tenantName}` : ''}
                            </Typography>
                            <Typography sx={{ display: { xs: 'block', md: 'none' }, fontSize: 12, color: tokens.slate[500] }}>
                              {type}
                              {r.description ? ` · ${r.description}` : ''}
                            </Typography>
                          </TableCell>
                          <TableCell sx={{ display: SHOW_MD }}>
                            <Typography sx={{ fontSize: 13 }}>{type}</Typography>
                          </TableCell>
                          <TableCell sx={{ display: SHOW_LG }}>
                            <Typography sx={{ fontSize: 13, color: tokens.slate[700] }}>{r.description || '—'}</Typography>
                          </TableCell>
                          <TableCell align="right">
                            <Typography sx={{ fontFamily: tokens.fontMono, fontSize: 13, fontWeight: 600 }}>{formatMoney(r.amountCents)}</Typography>
                          </TableCell>
                          <TableCell>
                            <StatusChip label={PAYMENT_STATUS_LABELS[status]} tone={PAYMENT_STATUS_TONE[status]} />
                          </TableCell>
                          <TableCell align="right">
                            <Button
                              size="small"
                              variant="outlined"
                              disabled={r.outstandingCents <= 0}
                              onClick={() => {
                                setPayError(null);
                                setPayTarget(r);
                              }}
                              sx={{ borderColor: tokens.slate[300], color: tokens.slate[700], whiteSpace: 'nowrap' }}
                            >
                              Record payment
                            </Button>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                </TableBody>
              </Table>
            </TableContainer>
            {!isLoading && total > 0 && (
              <TablePagination
                component="div"
                count={total}
                page={page}
                rowsPerPage={rowsPerPage}
                rowsPerPageOptions={[10, 25, 50]}
                onPageChange={(_e, p) => setPage(p)}
                onRowsPerPageChange={(e) => {
                  setRowsPerPage(Number(e.target.value));
                  setPage(0);
                }}
              />
            )}
          </Paper>
        </ErrorBoundary>
      )}

      <ChargeDialog
        open={newOpen}
        onClose={() => setNewOpen(false)}
        properties={properties}
        onSaved={() => {
          setNewOpen(false);
          setToast('Charge created');
        }}
      />

      <RecordPaymentDialog
        open={Boolean(payTarget)}
        title="Record payment"
        subtitle={payTarget ? `${payTarget.propertyName} · ${payTarget.unitName} — ${payTarget.description}` : undefined}
        outstandingCents={payTarget?.outstandingCents ?? 0}
        submitting={recordPayment.isPending}
        error={payError}
        onClose={() => setPayTarget(null)}
        onSubmit={(values) => {
          if (!payTarget) return;
          recordPayment.mutate(
            { transactionId: payTarget.id, values },
            {
              onSuccess: () => {
                setPayTarget(null);
                setToast('Payment recorded');
              },
              onError: (err) => setPayError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.'),
            },
          );
        }}
      />

      <Snackbar open={Boolean(toast)} autoHideDuration={3200} onClose={() => setToast(null)} anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}>
        {toast ? (
          <Alert onClose={() => setToast(null)} severity="success" variant="filled" sx={{ borderRadius: 999 }}>
            {toast}
          </Alert>
        ) : undefined}
      </Snackbar>
    </Box>
  );
}
