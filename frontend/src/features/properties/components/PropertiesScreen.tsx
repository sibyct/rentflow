import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { tokens } from '@/app/tokens';
import { ErrorBoundary, ErrorMessage } from '@/shared/components';
import type { PropertySortKey } from '../api/propertiesApi';
import { useProperties, usePropertiesTotalCount, useUpdatePropertyStatus } from '../hooks/usePropertiesQueries';
import type { PropertyRow, PropertyRowStatus, PropertyType } from '../mock/propertyRows';
import { PropertyFormModal } from './PropertyFormModal';
import { PropertiesToolbar } from './PropertiesToolbar';
import { PropertiesTable, type PropertiesEmptyState } from './PropertiesTable';

type Toast = { message: string; severity: 'success' | 'info' | 'error' };

/** null = closed; {} = create mode; {propertyId} = editing that property. */
type FormModalState = { propertyId?: string } | null;

export function PropertiesScreen() {
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  const [typeFilter, setTypeFilter] = useState<PropertyType | ''>('');
  const [statusFilter, setStatusFilter] = useState<PropertyRowStatus | ''>('Active');

  const [sortKey, setSortKey] = useState<PropertySortKey>('name');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);

  const [formModal, setFormModal] = useState<FormModalState>(null);
  const [toast, setToast] = useState<Toast | null>(null);
  const [newRowId, setNewRowId] = useState<string | null>(null);

  const { data, isLoading, isError, error, refetch } = useProperties({
    search: search || undefined,
    type: typeFilter,
    status: statusFilter,
    sort: sortKey,
    order: sortDir,
    limit: rowsPerPage,
    offset: page * rowsPerPage,
  });
  const updateStatus = useUpdatePropertyStatus();

  const pageRows = data?.properties ?? [];
  const total = data?.total ?? 0;
  const isEmptyResult = !isLoading && total === 0;

  // Empty-state diagnosis: only fires the extra (cheap, limit=1) queries
  // once the main list has actually come back empty, so the common case
  // (there are results) never pays for them.
  const { data: portfolioTotal } = usePropertiesTotalCount({}, { enabled: isEmptyResult });
  const hasAnyPropertyEver = (portfolioTotal ?? 0) > 0;

  // If status is the *only* filter narrowing the result (no search, no
  // type), it's worth checking whether lifting just that one filter
  // would reveal the rows — the classic "default filter is silently
  // hiding everything" trap, most often hit by a brand-new account
  // whose only properties are "Onboarding" while the view defaults to
  // "Active".
  const statusMightBeCulprit = isEmptyResult && hasAnyPropertyEver && Boolean(statusFilter) && !search && !typeFilter;
  const { data: countIgnoringStatus } = usePropertiesTotalCount(
    { search: search || undefined, type: typeFilter },
    { enabled: statusMightBeCulprit },
  );

  const emptyState: PropertiesEmptyState = !isEmptyResult
    ? { kind: 'none' }
    : !hasAnyPropertyEver
      ? { kind: 'first-time' }
      : statusMightBeCulprit && (countIgnoringStatus ?? 0) > 0
        ? { kind: 'status-culprit', statusLabel: statusFilter, count: countIgnoringStatus ?? 0 }
        : { kind: 'filtered' };

  function handleSort(key: PropertySortKey) {
    if (sortKey === key) setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    else {
      setSortKey(key);
      setSortDir('asc');
    }
    setPage(0);
  }

  function toggleRow(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }
  function toggleSelectAllOnPage() {
    const pageIds = pageRows.map((p) => p.id);
    const allPageSelected = pageIds.length > 0 && pageIds.every((id) => selected.has(id));
    setSelected((prev) => {
      const next = new Set(prev);
      if (allPageSelected) pageIds.forEach((id) => next.delete(id));
      else pageIds.forEach((id) => next.add(id));
      return next;
    });
  }

  /** Full reset: search, type, and status all back to their defaults. */
  function resetView() {
    setSearch('');
    setTypeFilter('');
    setStatusFilter('');
    setPage(0);
  }

  /** Scoped reset: lifts only the status filter, leaving search/type as they were. */
  function clearStatusFilter() {
    setStatusFilter('');
    setPage(0);
  }

  function bulkArchive() {
    const ids = Array.from(selected);
    updateStatus.mutate(
      { ids, status: 'Archived' },
      {
        onSuccess: (n) => {
          setSelected(new Set());
          setToast({ message: `Archived ${n} ${n === 1 ? 'property' : 'properties'}`, severity: 'success' });
        },
        onError: (err) => setToast({ message: errorMessage(err), severity: 'error' }),
      },
    );
  }
  function bulkExport() {
    setToast({ message: `Exported ${selected.size} ${selected.size === 1 ? 'property' : 'properties'}`, severity: 'success' });
  }

  function toggleArchive(p: PropertyRow) {
    const nextStatus: PropertyRowStatus = p.status === 'Archived' ? 'Active' : 'Archived';
    updateStatus.mutate(
      { ids: [p.id], status: nextStatus },
      {
        onSuccess: () => setToast({ message: `${nextStatus === 'Archived' ? 'Archived' : 'Restored'} ${p.name}`, severity: 'success' }),
        onError: (err) => setToast({ message: errorMessage(err), severity: 'error' }),
      },
    );
  }

  function handleSaved(row: PropertyRow, { addUnitsNow, isEdit }: { addUnitsNow: boolean; isEdit: boolean }) {
    if (!isEdit) {
      setStatusFilter(''); // reveal the new Onboarding row immediately
      setPage(0);
    }
    setNewRowId(row.id);
    setFormModal(null);
    setToast({
      message: isEdit ? 'Property updated successfully' : addUnitsNow ? 'Property added — opening unit setup…' : 'Property added successfully',
      severity: 'success',
    });
    window.setTimeout(() => setNewRowId(null), 2400);
  }

  return (
    <Box>
      <PropertiesToolbar
        search={search}
        onSearchChange={(v) => {
          setSearch(v);
          setPage(0);
        }}
        typeFilter={typeFilter}
        onTypeFilterChange={(v) => {
          setTypeFilter(v);
          setPage(0);
        }}
        statusFilter={statusFilter}
        onStatusFilterChange={(v) => {
          setStatusFilter(v);
          setPage(0);
        }}
        onExport={() => setToast({ message: 'Exporting properties as CSV…', severity: 'info' })}
        onAddProperty={() => setFormModal({})}
      />

      {selected.size > 0 && (
        // Sticky, not just inline: with 25 rows/page the table can run
        // well past the viewport, and this bar would otherwise scroll
        // out of view the moment you scroll down to work a selection
        // made further down the list.
        <Stack
          direction="row"
          sx={{
            position: 'sticky',
            bottom: 16,
            zIndex: 5,
            alignItems: 'center',
            gap: 2,
            bgcolor: tokens.slate[900],
            color: '#fff',
            borderRadius: `${tokens.radiusControl}px`,
            px: 2,
            py: 1.25,
            mb: 2,
            boxShadow: tokens.shadowPopover,
          }}
        >
          <Typography sx={{ fontSize: 13 }}>
            <strong>{selected.size}</strong> selected
          </Typography>
          <Stack direction="row" spacing={1} sx={{ ml: 'auto' }}>
            <Button size="small" onClick={bulkExport} sx={{ color: '#fff', bgcolor: 'rgba(255,255,255,0.12)', '&:hover': { bgcolor: 'rgba(255,255,255,0.22)' } }}>
              Export
            </Button>
            <Button size="small" onClick={bulkArchive} disabled={updateStatus.isPending} sx={{ color: '#fff', bgcolor: 'rgba(255,255,255,0.12)', '&:hover': { bgcolor: 'rgba(255,255,255,0.22)' } }}>
              Archive
            </Button>
            <Button size="small" onClick={() => setSelected(new Set())} sx={{ color: '#fff', textDecoration: 'underline' }}>
              Clear
            </Button>
          </Stack>
        </Stack>
      )}

      {isError ? (
        // A failed fetch (network error, 500, ...) is expected/recoverable
        // and goes through QueryState-style handling with a Retry that
        // just refetches — it never throws, so it doesn't need a boundary.
        <ErrorMessage error={error} onRetry={() => void refetch()} />
      ) : (
        // A genuine render exception inside the table (bad row data, a
        // bug in a cell renderer, ...) is a different failure mode — it
        // throws, and only an error boundary can catch it. Scoped to
        // just the table: the header, filter bar, and bulk-actions bar
        // above stay fully rendered and interactive either way.
        <ErrorBoundary>
          <PropertiesTable
            loading={isLoading}
            pageRows={pageRows}
            total={total}
            emptyState={emptyState}
            selected={selected}
            onToggleRow={toggleRow}
            onToggleSelectAllOnPage={toggleSelectAllOnPage}
            sortKey={sortKey}
            sortDir={sortDir}
            onSort={handleSort}
            newRowId={newRowId}
            page={page}
            rowsPerPage={rowsPerPage}
            onPageChange={setPage}
            onRowsPerPageChange={(n) => {
              setRowsPerPage(n);
              setPage(0);
            }}
            onRowClick={(p) => navigate(`/properties/${p.id}`)}
            onEditRow={(p) => setFormModal({ propertyId: p.id })}
            onViewDetails={(p) => navigate(`/properties/${p.id}`)}
            onToggleArchive={toggleArchive}
            onAddProperty={() => setFormModal({})}
            onResetView={resetView}
            onClearStatusFilter={clearStatusFilter}
          />
        </ErrorBoundary>
      )}

      {/* The "add units now?" prompt is no longer a separate dialog that
          pops up after this one closes — it's the modal's own final
          step (PropertyFormModal's SuccessScreen, create mode only), so
          the whole create flow reads as one continuous thing instead of
          modal-closes-then-another-modal-opens. The same modal also
          handles Edit (see propertyId) — reusing the wizard rather than
          building a separate single-page edit form. */}
      <PropertyFormModal
        open={Boolean(formModal)}
        propertyId={formModal?.propertyId}
        onClose={() => setFormModal(null)}
        onSaved={handleSaved}
        onArchiveToggled={(message) => setToast({ message, severity: 'success' })}
      />

      <Snackbar open={Boolean(toast)} autoHideDuration={3600} onClose={() => setToast(null)} anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}>
        {toast ? (
          <Alert onClose={() => setToast(null)} severity={toast.severity} variant="filled" sx={{ borderRadius: 999 }}>
            {toast.message}
          </Alert>
        ) : undefined}
      </Snackbar>
    </Box>
  );
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'Something went wrong. Please try again.';
}
