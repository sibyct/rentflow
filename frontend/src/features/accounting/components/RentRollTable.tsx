import { Link as RouterLink } from 'react-router-dom';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TablePagination from '@mui/material/TablePagination';
import TableRow from '@mui/material/TableRow';
import TableSortLabel from '@mui/material/TableSortLabel';
import Typography from '@mui/material/Typography';
import RequestQuoteOutlined from '@mui/icons-material/RequestQuoteOutlined';
import SearchOffOutlined from '@mui/icons-material/SearchOffOutlined';
import { tokens } from '@/app/tokens';
import { EmptyState, StatusChip, TableSkeletonRows, type TableSkeletonColumn } from '@/shared/components';
import { formatDate, formatMoney, formatMonth } from '@/shared/lib/format';
import type { RentRollSortKey } from '../api/accountingApi';
import { PAYMENT_STATUS_LABELS, PAYMENT_STATUS_TONE, type RentRollRow } from '../types';

// Below each breakpoint a column drops out of the table and its content
// folds into the first cell instead (see `Summary`), so the table always
// fits its container — no horizontal scroll at any width.
const SHOW_MD = { xs: 'none', md: 'table-cell' } as const;
const SHOW_LG = { xs: 'none', lg: 'table-cell' } as const;
const SHOW_XL = { xs: 'none', xl: 'table-cell' } as const;

interface Column {
  label: string;
  sortKey?: RentRollSortKey;
  align?: 'left' | 'right';
  display?: typeof SHOW_MD | typeof SHOW_LG | typeof SHOW_XL;
}

const COLUMNS: Column[] = [
  { label: 'Property / Unit', sortKey: 'property' },
  { label: 'Tenant', display: SHOW_MD },
  { label: 'Rent', align: 'right', display: SHOW_LG },
  { label: 'Due', sortKey: 'dueOn', display: SHOW_MD },
  { label: 'Paid', align: 'right', display: SHOW_LG },
  { label: 'Outstanding', sortKey: 'outstanding', align: 'right' },
  { label: 'Status' },
  { label: 'Last payment', display: SHOW_XL },
];

const SKELETON_COLUMNS: TableSkeletonColumn[] = [
  { shapes: [{ width: '70%' }, { width: '45%' }] },
  { shapes: [{ width: '60%' }] },
  { align: 'right', shapes: [{ width: 60 }] },
  { shapes: [{ width: '55%' }] },
  { align: 'right', shapes: [{ width: 60 }] },
  { align: 'right', shapes: [{ width: 70 }] },
  { shapes: [{ variant: 'rounded', width: 64, height: 22 }] },
  { shapes: [{ width: '55%' }] },
  { shapes: [{ variant: 'rounded', width: 96, height: 30 }] },
];

const mono = { fontFamily: tokens.fontMono, fontSize: 13 } as const;

export type RentRollEmptyState = { kind: 'none' } | { kind: 'first-time' } | { kind: 'filtered' };

interface RentRollTableProps {
  loading: boolean;
  rows: RentRollRow[];
  total: number;
  emptyState: RentRollEmptyState;
  sortKey: RentRollSortKey;
  sortDir: 'asc' | 'desc';
  onSort: (key: RentRollSortKey) => void;
  page: number;
  rowsPerPage: number;
  onPageChange: (page: number) => void;
  onRowsPerPageChange: (rowsPerPage: number) => void;
  onResetView: () => void;
  onRecordPayment: (row: RentRollRow) => void;
}

/** What the hidden columns would have shown, folded under the property/unit on narrow screens. */
function Summary({ row }: { row: RentRollRow }) {
  return (
    <Box sx={{ display: { xs: 'block', md: 'none' }, mt: 0.25 }}>
      <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>
        {row.tenantName} · rent {formatMoney(row.rentCents)} · due {formatDate(row.dueOn)}
      </Typography>
      {row.paidCents > 0 && (
        <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>
          paid {formatMoney(row.paidCents)}
          {row.lastPaidOn ? ` on ${formatDate(row.lastPaidOn)}` : ''}
        </Typography>
      )}
    </Box>
  );
}

/** Rent Roll: one row per lease per billing month, with money right-aligned in mono and the balance including late fees and charges. */
export function RentRollTable({
  loading,
  rows,
  total,
  emptyState,
  sortKey,
  sortDir,
  onSort,
  page,
  rowsPerPage,
  onPageChange,
  onRowsPerPageChange,
  onResetView,
  onRecordPayment,
}: RentRollTableProps) {
  return (
    <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
      <TableContainer>
        <Table size="small">
          {(loading || total > 0) && (
            <TableHead>
              <TableRow>
                {COLUMNS.map((col) => (
                  <TableCell key={col.label} align={col.align ?? 'left'} sx={{ display: col.display }}>
                    {col.sortKey ? (
                      <TableSortLabel
                        active={sortKey === col.sortKey}
                        direction={sortKey === col.sortKey ? sortDir : 'asc'}
                        onClick={() => onSort(col.sortKey!)}
                        sx={{ '& .MuiTableSortLabel-icon': { opacity: sortKey === col.sortKey ? 1 : 0.4 } }}
                      >
                        {col.label}
                      </TableSortLabel>
                    ) : (
                      col.label
                    )}
                  </TableCell>
                ))}
                <TableCell />
              </TableRow>
            </TableHead>
          )}
          <TableBody>
            {loading && <TableSkeletonRows columns={SKELETON_COLUMNS} />}

            {!loading && emptyState.kind === 'first-time' && (
              <TableRow>
                <TableCell colSpan={9}>
                  <EmptyState
                    icon={RequestQuoteOutlined}
                    title="Nothing to collect yet"
                    description="Rent shows up here for every active lease once it is billed. Add a lease to a unit to get started."
                  />
                </TableCell>
              </TableRow>
            )}

            {!loading && emptyState.kind === 'filtered' && (
              <TableRow>
                <TableCell colSpan={9}>
                  <EmptyState
                    icon={SearchOffOutlined}
                    title="No rent rows match your filters"
                    description="Try a different month or status, or reset the view."
                    action={
                      <Button variant="outlined" onClick={onResetView} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
                        Reset view
                      </Button>
                    }
                  />
                </TableCell>
              </TableRow>
            )}

            {!loading &&
              rows.map((r) => (
                <TableRow key={`${r.leaseId}-${r.period}`} hover>
                  <TableCell>
                    <Link component={RouterLink} to={`/leases/${r.leaseId}`} sx={{ fontSize: 13, fontWeight: 600 }}>
                      {r.propertyName}
                    </Link>
                    <Typography sx={{ fontSize: 12.5, color: tokens.slate[600] }}>
                      {r.unitName} · {formatMonth(r.period)}
                    </Typography>
                    <Summary row={r} />
                  </TableCell>
                  <TableCell sx={{ display: SHOW_MD }}>
                    <Typography sx={{ fontSize: 13 }}>{r.tenantName}</Typography>
                  </TableCell>
                  <TableCell align="right" sx={{ display: SHOW_LG }}>
                    <Typography sx={mono}>{formatMoney(r.rentCents)}</Typography>
                  </TableCell>
                  <TableCell sx={{ display: SHOW_MD }}>
                    <Typography sx={{ fontSize: 13 }}>{formatDate(r.dueOn)}</Typography>
                  </TableCell>
                  <TableCell align="right" sx={{ display: SHOW_LG }}>
                    <Typography sx={mono}>{formatMoney(r.paidCents)}</Typography>
                  </TableCell>
                  <TableCell align="right">
                    <Typography sx={{ ...mono, fontWeight: 600, color: r.outstandingCents > 0 ? tokens.slate[900] : tokens.slate[400] }}>
                      {formatMoney(r.outstandingCents)}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <StatusChip label={PAYMENT_STATUS_LABELS[r.status]} tone={PAYMENT_STATUS_TONE[r.status]} />
                  </TableCell>
                  <TableCell sx={{ display: SHOW_XL }}>
                    <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>{formatDate(r.lastPaidOn)}</Typography>
                  </TableCell>
                  <TableCell align="right">
                    <Button
                      size="small"
                      variant="outlined"
                      disabled={r.outstandingCents <= 0}
                      onClick={() => onRecordPayment(r)}
                      sx={{ borderColor: tokens.slate[300], color: tokens.slate[700], whiteSpace: 'nowrap' }}
                    >
                      Record payment
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </TableContainer>

      {!loading && total > 0 && (
        <TablePagination
          component="div"
          count={total}
          page={page}
          rowsPerPage={rowsPerPage}
          rowsPerPageOptions={[10, 25, 50]}
          onPageChange={(_e, p) => onPageChange(p)}
          onRowsPerPageChange={(e) => onRowsPerPageChange(Number(e.target.value))}
        />
      )}
    </Paper>
  );
}
