import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import InputAdornment from '@mui/material/InputAdornment';
import MenuItem from '@mui/material/MenuItem';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import AddOutlined from '@mui/icons-material/AddOutlined';
import SearchOutlined from '@mui/icons-material/SearchOutlined';
import { tokens } from '@/app/tokens';
import { useProperties } from '@/features/properties/hooks/usePropertiesQueries';
import { useVendors } from '@/features/vendors/hooks/useVendorsQueries';
import { ErrorBoundary, ErrorMessage } from '@/shared/components';
import type { ExpenseSortKey } from '../api/accountingApi';
import { useExpenses } from '../hooks/useAccountingQueries';
import {
  EXPENSE_CATEGORIES,
  EXPENSE_CATEGORY_LABELS,
  EXPENSE_STATUSES,
  EXPENSE_STATUS_LABELS,
  type ExpenseCategory,
  type ExpenseStatus,
} from '../types';
import { ExpenseDrawer } from './ExpenseDrawer';
import { ExpensesTable, type ExpensesEmptyState } from './ExpensesTable';

/** /accounting/expenses — every expense across the portfolio, filterable, with add/edit in a drawer. */
export function ExpensesScreen() {
  const [searchParams] = useSearchParams();

  const [search, setSearch] = useState('');
  const [propertyFilter, setPropertyFilter] = useState('');
  const [categoryFilter, setCategoryFilter] = useState<ExpenseCategory | ''>('');
  const [vendorFilter, setVendorFilter] = useState('');
  // Seeded from ?status=unpaid — the accounting dashboard's Upcoming Expenses card links here.
  const [statusFilter, setStatusFilter] = useState<ExpenseStatus | ''>(
    (searchParams.get('status') as ExpenseStatus | null) ?? '',
  );
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [sortKey, setSortKey] = useState<ExpenseSortKey>('incurredOn');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [drawer, setDrawer] = useState<{ open: boolean; expenseId?: string }>({ open: false });
  const [toast, setToast] = useState<string | null>(null);

  const { data: propertiesData } = useProperties({ limit: 100, offset: 0 });
  const properties = propertiesData?.properties ?? [];
  const { data: vendorsData } = useVendors({ limit: 100, offset: 0 });
  const vendors = vendorsData?.vendors ?? [];

  const { data, isLoading, isError, error, refetch } = useExpenses({
    search: search || undefined,
    propertyId: propertyFilter || undefined,
    category: categoryFilter || undefined,
    vendorId: vendorFilter || undefined,
    status: statusFilter || undefined,
    from: from || undefined,
    to: to || undefined,
    sort: sortKey,
    order: sortDir,
    limit: rowsPerPage,
    offset: page * rowsPerPage,
  });

  const rows = data?.rows ?? [];
  const total = data?.total ?? 0;
  const isFiltered = Boolean(search || propertyFilter || categoryFilter || vendorFilter || statusFilter || from || to);
  const emptyState: ExpensesEmptyState =
    isLoading || total > 0 ? { kind: 'none' } : isFiltered ? { kind: 'filtered' } : { kind: 'first-time' };

  function handleSort(key: ExpenseSortKey) {
    if (sortKey === key) setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    else {
      setSortKey(key);
      setSortDir('asc');
    }
    setPage(0);
  }

  function resetView() {
    setSearch('');
    setPropertyFilter('');
    setCategoryFilter('');
    setVendorFilter('');
    setStatusFilter('');
    setFrom('');
    setTo('');
    setPage(0);
  }

  const filter = (setter: (v: string) => void) => (e: { target: { value: string } }) => {
    setter(e.target.value);
    setPage(0);
  };

  return (
    <Box>
      <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center', flexWrap: 'wrap', mb: 2, rowGap: 1.25 }}>
        <TextField
          size="small"
          placeholder="Search expenses"
          value={search}
          onChange={filter(setSearch)}
          sx={{ flex: '1 1 180px', maxWidth: 260 }}
          slotProps={{
            input: {
              startAdornment: (
                <InputAdornment position="start">
                  <SearchOutlined sx={{ fontSize: 18, color: tokens.slate[400] }} />
                </InputAdornment>
              ),
            },
          }}
        />
        <TextField select size="small" slotProps={{ select: { displayEmpty: true } }} value={propertyFilter} onChange={filter(setPropertyFilter)} sx={{ minWidth: 170 }}>
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
          value={categoryFilter}
          onChange={filter((v) => setCategoryFilter(v as ExpenseCategory | ''))}
          sx={{ minWidth: 160 }}
        >
          <MenuItem value="">Category: All</MenuItem>
          {EXPENSE_CATEGORIES.map((c) => (
            <MenuItem key={c} value={c}>
              {EXPENSE_CATEGORY_LABELS[c]}
            </MenuItem>
          ))}
        </TextField>
        <TextField select size="small" slotProps={{ select: { displayEmpty: true } }} value={vendorFilter} onChange={filter(setVendorFilter)} sx={{ minWidth: 160 }}>
          <MenuItem value="">Vendor: All</MenuItem>
          {vendors.map((v) => (
            <MenuItem key={v.id} value={v.id}>
              {v.companyName}
            </MenuItem>
          ))}
        </TextField>
        <TextField
          select
          size="small"
          slotProps={{ select: { displayEmpty: true } }}
          value={statusFilter}
          onChange={filter((v) => setStatusFilter(v as ExpenseStatus | ''))}
          sx={{ minWidth: 140 }}
        >
          <MenuItem value="">Status: All</MenuItem>
          {EXPENSE_STATUSES.map((s) => (
            <MenuItem key={s} value={s}>
              {EXPENSE_STATUS_LABELS[s]}
            </MenuItem>
          ))}
        </TextField>
        <TextField size="small" type="date" label="From" value={from} onChange={filter(setFrom)} slotProps={{ inputLabel: { shrink: true } }} sx={{ width: 150 }} />
        <TextField size="small" type="date" label="To" value={to} onChange={filter(setTo)} slotProps={{ inputLabel: { shrink: true } }} sx={{ width: 150 }} />

        <Box sx={{ flex: 1 }} />

        <Button size="small" variant="contained" startIcon={<AddOutlined />} onClick={() => setDrawer({ open: true })} sx={{ flexShrink: 0 }}>
          Add expense
        </Button>
      </Stack>

      {isError ? (
        <ErrorMessage error={error} onRetry={() => void refetch()} />
      ) : (
        <ErrorBoundary>
          <ExpensesTable
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
            onAdd={() => setDrawer({ open: true })}
            onResetView={resetView}
            onOpen={(r) => setDrawer({ open: true, expenseId: r.id })}
          />
        </ErrorBoundary>
      )}

      <ExpenseDrawer
        open={drawer.open}
        expenseId={drawer.expenseId}
        properties={properties}
        onClose={() => setDrawer({ open: false })}
        onSaved={(message) => {
          setDrawer({ open: false });
          setToast(message);
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
