import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
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
import type { VendorSortKey } from '../api/vendorsApi';
import { useDeleteVendor, useVendors } from '../hooks/useVendorsQueries';
import type { VendorRow, WorkOrderCategory } from '../types';
import { VendorFormDialog } from './VendorFormDialog';
import { VendorsTable, type VendorsEmptyState } from './VendorsTable';
import { VendorsToolbar } from './VendorsToolbar';

/**
 * The Vendors list page — every vendor the owner has, filterable/
 * sortable/searchable. Viewing a row navigates to its full profile
 * page (/vendors/:id): like Leases, a vendor has enough sections
 * (contact, compliance, billing, work order history, spend) to warrant
 * a dedicated page rather than an inline drawer.
 */
export function VendorsScreen() {
  const navigate = useNavigate();

  const [search, setSearch] = useState('');
  const [categoryFilter, setCategoryFilter] = useState<WorkOrderCategory | ''>('');
  const [activeFilter, setActiveFilter] = useState<'active' | 'inactive' | ''>('');
  const [sortKey, setSortKey] = useState<VendorSortKey>('name');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [addOpen, setAddOpen] = useState(false);
  const [editVendor, setEditVendor] = useState<VendorRow | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<VendorRow | null>(null);
  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);

  const deleteVendor = useDeleteVendor();

  const { data: propertiesData } = useProperties({ limit: 100, offset: 0 });
  const properties = propertiesData?.properties ?? [];

  const { data, isLoading, isError, error, refetch } = useVendors({
    search: search || undefined,
    category: categoryFilter || undefined,
    active: activeFilter ? activeFilter === 'active' : undefined,
    sort: sortKey,
    order: sortDir,
    limit: rowsPerPage,
    offset: page * rowsPerPage,
  });

  const vendors = data?.vendors ?? [];
  const total = data?.total ?? 0;
  const isFiltered = Boolean(search || categoryFilter || activeFilter);
  const emptyState: VendorsEmptyState = isLoading || total > 0 ? { kind: 'none' } : isFiltered ? { kind: 'filtered' } : { kind: 'first-time' };

  function handleSort(key: VendorSortKey) {
    if (sortKey === key) setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    else {
      setSortKey(key);
      setSortDir('asc');
    }
    setPage(0);
  }

  function resetView() {
    setSearch('');
    setCategoryFilter('');
    setActiveFilter('');
    setPage(0);
  }

  return (
    <Box>
      <VendorsToolbar
        search={search}
        onSearchChange={(v) => {
          setSearch(v);
          setPage(0);
        }}
        categoryFilter={categoryFilter}
        onCategoryFilterChange={(v) => {
          setCategoryFilter(v);
          setPage(0);
        }}
        activeFilter={activeFilter}
        onActiveFilterChange={(v) => {
          setActiveFilter(v);
          setPage(0);
        }}
        onAddVendor={() => setAddOpen(true)}
      />

      {isError ? (
        <ErrorMessage error={error} onRetry={() => void refetch()} />
      ) : (
        <ErrorBoundary>
          <VendorsTable
            loading={isLoading}
            vendors={vendors}
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
            onAddVendor={() => setAddOpen(true)}
            onResetView={resetView}
            onViewVendor={(v) => navigate(`/vendors/${v.id}`)}
            onEditVendor={setEditVendor}
            onDeleteVendor={setDeleteTarget}
            onOpenJobsClick={(v) => navigate(`/maintenance?vendor_id=${v.id}&vendor_name=${encodeURIComponent(v.companyName)}`)}
          />
        </ErrorBoundary>
      )}

      <VendorFormDialog
        open={addOpen}
        onClose={() => setAddOpen(false)}
        properties={properties}
        onSaved={() => {
          setAddOpen(false);
          setToast({ message: 'Vendor added', severity: 'success' });
        }}
      />

      <VendorFormDialog
        open={Boolean(editVendor)}
        onClose={() => setEditVendor(null)}
        properties={properties}
        vendorId={editVendor?.id}
        onSaved={() => {
          setEditVendor(null);
          setToast({ message: 'Vendor updated', severity: 'success' });
        }}
      />

      <Dialog open={Boolean(deleteTarget)} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete this vendor?</DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>
            {deleteTarget ? `"${deleteTarget.companyName}"` : 'This vendor'} will be permanently removed. Work orders already assigned to it keep their history but lose the vendor link. This can&apos;t be undone.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 0 }}>
          <Button variant="text" onClick={() => setDeleteTarget(null)} sx={{ color: tokens.slate[600] }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color="error"
            disabled={deleteVendor.isPending}
            onClick={() => {
              if (!deleteTarget) return;
              deleteVendor.mutate(deleteTarget.id, {
                onSuccess: () => {
                  setToast({ message: 'Vendor removed', severity: 'success' });
                  setDeleteTarget(null);
                },
                onError: () => setToast({ message: 'Something went wrong. Please try again.', severity: 'error' }),
              });
            }}
          >
            Delete vendor
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
