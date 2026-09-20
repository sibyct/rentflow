import { useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import Snackbar from '@mui/material/Snackbar';
import Typography from '@mui/material/Typography';
import { tokens } from '@/app/tokens';
import { useProperties } from '@/features/properties/hooks/usePropertiesQueries';
import { useUnitsPortfolio } from '@/features/units/hooks/useUnitsQueries';
import { ErrorBoundary, ErrorMessage } from '@/shared/components';
import type { LeasePortfolioSortKey } from '../api/leasesApi';
import { useDeleteLease, useLeasesPortfolio } from '../hooks/useLeasesQueries';
import type { LeaseDisplayStatus, LeaseRowWithUnitProperty } from '../types';
import { LeaseFormDialog } from './LeaseFormDialog';
import { LeasesTable, type LeasesPortfolioEmptyState } from './LeasesTable';
import { LeasesToolbar } from './LeasesToolbar';

/**
 * The portfolio-wide zoom level of the Leases feature — every lease
 * across every property, filterable/sortable/searchable. Viewing a row
 * navigates to its full detail page (/leases/:id): a lease has more
 * sections (parties, financials, renewal/termination) than a unit, so
 * unlike UnitsSection's inline drawer, this warrants a dedicated page.
 */
export function LeasesScreen() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const [search, setSearch] = useState('');
  const [propertyFilter, setPropertyFilter] = useState('');
  // Seeded once from ?status=expiring_soon — the Dashboard's "Leases
  // Expiring Soon" View all link lands here with.
  const [statusFilter, setStatusFilter] = useState<LeaseDisplayStatus | ''>(
    (searchParams.get('status') as LeaseDisplayStatus | null) ?? '',
  );
  const [sortKey, setSortKey] = useState<LeasePortfolioSortKey>('startDate');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [addOpen, setAddOpen] = useState(false);
  const [editLease, setEditLease] = useState<LeaseRowWithUnitProperty | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<LeaseRowWithUnitProperty | null>(null);
  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);

  const deleteLease = useDeleteLease(deleteTarget?.unitId ?? '');

  const { data: propertiesData } = useProperties({ limit: 100, offset: 0 });
  const properties = propertiesData?.properties ?? [];
  // Backs the Add Lease unit picker — every unit across the portfolio,
  // same 100-row cap as the Units page's own picker.
  const { data: unitsData } = useUnitsPortfolio({ limit: 100, offset: 0 });
  const units = unitsData?.units ?? [];

  const { data, isLoading, isError, error, refetch } = useLeasesPortfolio({
    search: search || undefined,
    propertyId: propertyFilter || undefined,
    status: statusFilter || undefined,
    sort: sortKey,
    order: sortDir,
    limit: rowsPerPage,
    offset: page * rowsPerPage,
  });

  const leases = data?.leases ?? [];
  const total = data?.total ?? 0;
  const isFiltered = Boolean(search || propertyFilter || statusFilter);
  const emptyState: LeasesPortfolioEmptyState =
    isLoading || total > 0 ? { kind: 'none' } : isFiltered ? { kind: 'filtered' } : { kind: 'first-time' };

  function handleSort(key: LeasePortfolioSortKey) {
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
    setStatusFilter('');
    setPage(0);
  }

  return (
    <Box>
      <LeasesToolbar
        search={search}
        onSearchChange={(v) => {
          setSearch(v);
          setPage(0);
        }}
        properties={properties}
        propertyFilter={propertyFilter}
        onPropertyFilterChange={(v) => {
          setPropertyFilter(v);
          setPage(0);
        }}
        statusFilter={statusFilter}
        onStatusFilterChange={(v) => {
          setStatusFilter(v);
          setPage(0);
        }}
        onAddLease={() => setAddOpen(true)}
      />

      {isError ? (
        <ErrorMessage error={error} onRetry={() => void refetch()} />
      ) : (
        <ErrorBoundary>
          <LeasesTable
            loading={isLoading}
            leases={leases}
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
            onAddLease={() => setAddOpen(true)}
            onResetView={resetView}
            onViewLease={(l) => navigate(`/leases/${l.id}`)}
            onEditLease={setEditLease}
            onDeleteLease={setDeleteTarget}
          />
        </ErrorBoundary>
      )}

      <LeaseFormDialog
        open={addOpen}
        onClose={() => setAddOpen(false)}
        units={units}
        onSaved={() => {
          setAddOpen(false);
          setToast({ message: 'Lease added', severity: 'success' });
        }}
      />

      <LeaseFormDialog
        open={Boolean(editLease)}
        onClose={() => setEditLease(null)}
        unitId={editLease?.unitId}
        leaseId={editLease?.id}
        onSaved={() => {
          setEditLease(null);
          setToast({ message: 'Lease updated', severity: 'success' });
        }}
      />

      <Dialog open={Boolean(deleteTarget)} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete this lease?</DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>
            {deleteTarget ? `The lease for "${deleteTarget.primaryResidentName}" at ${deleteTarget.propertyName} / ${deleteTarget.unitName}` : 'This lease'} will be permanently removed. This can&apos;t be undone.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 0 }}>
          <Button variant="text" onClick={() => setDeleteTarget(null)} sx={{ color: tokens.slate[600] }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color="error"
            disabled={deleteLease.isPending}
            onClick={() => {
              if (!deleteTarget) return;
              deleteLease.mutate(deleteTarget.id, {
                onSuccess: () => {
                  setToast({ message: 'Lease removed', severity: 'success' });
                  setDeleteTarget(null);
                },
                onError: () => setToast({ message: 'Something went wrong. Please try again.', severity: 'error' }),
              });
            }}
          >
            Delete lease
          </Button>
        </DialogActions>
      </Dialog>

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
