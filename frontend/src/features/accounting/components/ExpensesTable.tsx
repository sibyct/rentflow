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
import AddOutlined from '@mui/icons-material/AddOutlined';
import ReceiptLongOutlined from '@mui/icons-material/ReceiptLongOutlined';
import SearchOffOutlined from '@mui/icons-material/SearchOffOutlined';
import { tokens } from '@/app/tokens';
import { EmptyState, StatusChip, TableSkeletonRows, type TableSkeletonColumn } from '@/shared/components';
import { formatDate, formatMoney } from '@/shared/lib/format';
import type { ExpenseSortKey } from '../api/accountingApi';
import {
  EXPENSE_CATEGORY_LABELS,
  EXPENSE_STATUS_LABELS,
  EXPENSE_STATUS_TONE,
  type ExpenseStatus,
  type LedgerRow,
} from '../types';

const SHOW_MD = { xs: 'none', md: 'table-cell' } as const;
const SHOW_LG = { xs: 'none', lg: 'table-cell' } as const;
const SHOW_XL = { xs: 'none', xl: 'table-cell' } as const;

interface Column {
  label: string;
  sortKey?: ExpenseSortKey;
  align?: 'left' | 'right';
  display?: typeof SHOW_MD | typeof SHOW_LG | typeof SHOW_XL;
}

const COLUMNS: Column[] = [
  { label: 'Date', sortKey: 'incurredOn' },
  { label: 'Property', sortKey: 'property' },
  { label: 'Category', sortKey: 'category', display: SHOW_MD },
  { label: 'Vendor', display: SHOW_LG },
  { label: 'Amount', sortKey: 'amount', align: 'right' },
  { label: 'Status' },
  { label: 'Work order', display: SHOW_XL },
];

const SKELETON_COLUMNS: TableSkeletonColumn[] = [
  { shapes: [{ width: '70%' }] },
  { shapes: [{ width: '70%' }, { width: '40%' }] },
  { shapes: [{ width: '60%' }] },
  { shapes: [{ width: '60%' }] },
  { align: 'right', shapes: [{ width: 70 }] },
  { shapes: [{ variant: 'rounded', width: 64, height: 22 }] },
  { shapes: [{ width: '50%' }] },
];

export type ExpensesEmptyState = { kind: 'none' } | { kind: 'first-time' } | { kind: 'filtered' };

interface ExpensesTableProps {
  loading: boolean;
  rows: LedgerRow[];
  total: number;
  emptyState: ExpensesEmptyState;
  sortKey: ExpenseSortKey;
  sortDir: 'asc' | 'desc';
  onSort: (key: ExpenseSortKey) => void;
  page: number;
  rowsPerPage: number;
  onPageChange: (page: number) => void;
  onRowsPerPageChange: (rowsPerPage: number) => void;
  onAdd: () => void;
  onResetView: () => void;
  onOpen: (row: LedgerRow) => void;
}

/** Expenses list. Secondary columns collapse into the first cells below md/lg/xl so the table never needs a horizontal scrollbar. */
export function ExpensesTable({
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
  onAdd,
  onResetView,
  onOpen,
}: ExpensesTableProps) {
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
              </TableRow>
            </TableHead>
          )}
          <TableBody>
            {loading && <TableSkeletonRows columns={SKELETON_COLUMNS} />}

            {!loading && emptyState.kind === 'first-time' && (
              <TableRow>
                <TableCell colSpan={7}>
                  <EmptyState
                    icon={ReceiptLongOutlined}
                    title="No expenses yet"
                    description="Track bills, repairs and recurring costs here. Completed work orders with an actual cost show up automatically."
                    action={
                      <Button variant="contained" startIcon={<AddOutlined />} onClick={onAdd}>
                        Add expense
                      </Button>
                    }
                  />
                </TableCell>
              </TableRow>
            )}

            {!loading && emptyState.kind === 'filtered' && (
              <TableRow>
                <TableCell colSpan={7}>
                  <EmptyState
                    icon={SearchOffOutlined}
                    title="No expenses match your filters"
                    description="Try different filters, or reset the view to see everything."
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
              rows.map((r) => {
                const status = r.status as ExpenseStatus;
                const category = r.category ? EXPENSE_CATEGORY_LABELS[r.category] : '—';
                return (
                  <TableRow key={r.id} hover onClick={() => onOpen(r)} sx={{ cursor: 'pointer' }}>
                    <TableCell>
                      <Typography sx={{ fontSize: 13 }}>{formatDate(r.incurredOn)}</Typography>
                    </TableCell>
                    <TableCell>
                      <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{r.propertyName}</Typography>
                      <Typography sx={{ display: { xs: 'block', md: 'none' }, fontSize: 12, color: tokens.slate[500] }}>
                        {category}
                        {r.vendorName ? ` · ${r.vendorName}` : ''}
                      </Typography>
                      {r.unitName && <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>{r.unitName}</Typography>}
                    </TableCell>
                    <TableCell sx={{ display: SHOW_MD }}>
                      <Typography sx={{ fontSize: 13 }}>{category}</Typography>
                    </TableCell>
                    <TableCell sx={{ display: SHOW_LG }}>
                      <Typography sx={{ fontSize: 13, color: tokens.slate[700] }}>{r.vendorName || '—'}</Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Typography sx={{ fontFamily: tokens.fontMono, fontSize: 13, fontWeight: 600 }}>{formatMoney(r.amountCents)}</Typography>
                    </TableCell>
                    <TableCell>
                      <StatusChip label={EXPENSE_STATUS_LABELS[status]} tone={EXPENSE_STATUS_TONE[status]} />
                    </TableCell>
                    <TableCell sx={{ display: SHOW_XL }} onClick={(e) => e.stopPropagation()}>
                      {r.workOrderId ? (
                        <Link component={RouterLink} to="/maintenance" sx={{ fontSize: 13 }}>
                          {r.workOrderTitle || 'Work order'}
                        </Link>
                      ) : (
                        <Box component="span" sx={{ color: tokens.slate[400] }}>
                          —
                        </Box>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}
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
