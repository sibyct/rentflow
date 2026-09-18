import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
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
import { ErrorBoundary, ErrorMessage } from '@/shared/components';
import type { UnitPortfolioSortKey } from '../api/unitsApi';
import { useDeleteUnit, useUnitsPortfolio } from '../hooks/useUnitsQueries';
import type { UnitRowWithProperty, UnitStatus, UnitType } from '../types';
import { GlobalUnitsTable, type UnitsPortfolioEmptyState } from './GlobalUnitsTable';
import { UnitDetailDrawer } from './UnitDetailDrawer';
import { UnitFormDialog } from './UnitFormDialog';
import { UnitsToolbar } from './UnitsToolbar';

/**
 * The portfolio-wide zoom level of the Units feature — every unit
 * across every property, filterable/sortable/searchable. Its sibling,
 * the property-scoped section on each property's detail page (see
 * UnitsSection), stays the fast contextual add/edit surface; this page
 * is for cross-property visibility, plus the one place "Add Unit"
 * genuinely needs a property picker since there's no page context to
 * infer it from.
 */
export function UnitsScreen() {
  const [searchParams, setSearchParams] = useSearchParams();

  const [search, setSearch] = useState('');
  // Seeded once from ?property=<id> — the deep-link UnitsSection's "View
  // in Units →" link lands here with.
  const [propertyFilter, setPropertyFilter] = useState(searchParams.get('property') ?? '');
  const [statusFilter, setStatusFilter] = useState<UnitStatus | ''>('');
  const [typeFilter, setTypeFilter] = useState<UnitType | ''>('');
  const [sortKey, setSortKey] = useState<UnitPortfolioSortKey>('propertyName');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [addOpen, setAddOpen] = useState(false);
  const [viewUnit, setViewUnit] = useState<UnitRowWithProperty | null>(null);
  const [editUnit, setEditUnit] = useState<UnitRowWithProperty | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<UnitRowWithProperty | null>(null);
  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);

  // Scoped to whichever unit is pending delete — a row's propertyId
  // varies across this portfolio-wide table, unlike UnitsSection where
  // it's fixed for the whole page, so the hook is re-created per target
  // rather than once with a constant propertyId.
  const deleteUnit = useDeleteUnit(deleteTarget?.propertyId ?? '');

  // Keeps the URL's ?property= in sync with the filter so this view
  // stays shareable/deep-linkable — not just a one-time seed on mount.
  useEffect(() => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        if (propertyFilter) next.set('property', propertyFilter);
        else next.delete('property');
        return next;
      },
      { replace: true },
    );
    // Only ever reacts to propertyFilter — setSearchParams is stable
    // from react-router and including it would just be noise here.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [propertyFilter]);

  // Backs both the Property filter dropdown and the Add Unit picker —
  // limit:100 is this API's max page size (see backend maxPageLimit),
  // so a portfolio past that size won't list every property here yet.
  const { data: propertiesData } = useProperties({ limit: 100, offset: 0 });
  const properties = propertiesData?.properties ?? [];

  const { data, isLoading, isError, error, refetch } = useUnitsPortfolio({
    search: search || undefined,
    propertyId: propertyFilter || undefined,
    status: statusFilter,
    type: typeFilter,
    sort: sortKey,
    order: sortDir,
    limit: rowsPerPage,
    offset: page * rowsPerPage,
  });

  const units = data?.units ?? [];
  const total = data?.total ?? 0;
  const isFiltered = Boolean(search || propertyFilter || statusFilter || typeFilter);
  const emptyState: UnitsPortfolioEmptyState =
    isLoading || total > 0 ? { kind: 'none' } : isFiltered ? { kind: 'filtered' } : { kind: 'first-time' };

  function handleSort(key: UnitPortfolioSortKey) {
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
    setTypeFilter('');
    setPage(0);
  }

  return (
    <Box>
      <UnitsToolbar
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
        typeFilter={typeFilter}
        onTypeFilterChange={(v) => {
          setTypeFilter(v);
          setPage(0);
        }}
        onAddUnit={() => setAddOpen(true)}
      />

      {isError ? (
        <ErrorMessage error={error} onRetry={() => void refetch()} />
      ) : (
        <ErrorBoundary>
          <GlobalUnitsTable
            loading={isLoading}
            units={units}
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
            onAddUnit={() => setAddOpen(true)}
            onResetView={resetView}
            onViewUnit={setViewUnit}
            onEditUnit={setEditUnit}
            onDeleteUnit={setDeleteTarget}
          />
        </ErrorBoundary>
      )}

      <UnitFormDialog
        open={addOpen}
        onClose={() => setAddOpen(false)}
        properties={properties}
        onSaved={() => {
          setAddOpen(false);
          setToast({ message: 'Unit added', severity: 'success' });
        }}
      />

      <UnitDetailDrawer
        unitId={viewUnit?.id ?? null}
        propertyId={viewUnit?.propertyId ?? ''}
        propertyName={viewUnit?.propertyName ?? ''}
        onClose={() => setViewUnit(null)}
        onEdit={() => {
          if (viewUnit) setEditUnit(viewUnit);
          setViewUnit(null);
        }}
        onDeleted={() => {
          setViewUnit(null);
          setToast({ message: 'Unit removed', severity: 'success' });
        }}
      />

      <UnitFormDialog
        open={Boolean(editUnit)}
        onClose={() => setEditUnit(null)}
        propertyId={editUnit?.propertyId}
        unitId={editUnit?.id}
        onSaved={() => {
          setEditUnit(null);
          setToast({ message: 'Unit updated', severity: 'success' });
        }}
      />

      <Dialog open={Boolean(deleteTarget)} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete this unit?</DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>
            {deleteTarget ? `"${deleteTarget.unitName}" at ${deleteTarget.propertyName}` : 'This unit'} will be permanently removed. This can&apos;t be undone.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 0 }}>
          <Button variant="text" onClick={() => setDeleteTarget(null)} sx={{ color: tokens.slate[600] }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color="error"
            disabled={deleteUnit.isPending}
            onClick={() => {
              if (!deleteTarget) return;
              deleteUnit.mutate(deleteTarget.id, {
                onSuccess: () => {
                  setToast({ message: 'Unit removed', severity: 'success' });
                  setDeleteTarget(null);
                },
                onError: () => setToast({ message: 'Something went wrong. Please try again.', severity: 'error' }),
              });
            }}
          >
            Delete unit
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
