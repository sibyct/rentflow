import { useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import AddOutlined from '@mui/icons-material/AddOutlined';
import ArrowForwardOutlined from '@mui/icons-material/ArrowForwardOutlined';
import { tokens } from '@/app/tokens';
import { ErrorBoundary, ErrorMessage } from '@/shared/components';
import type { WorkOrderSortKey } from '../api/workOrdersApi';
import { useDeleteWorkOrder, useWorkOrders } from '../hooks/useWorkOrdersQueries';
import type { WorkOrderRow } from '../types';
import { WorkOrderDrawer } from './WorkOrderDrawer';
import { WorkOrdersTable, type MaintenanceEmptyState } from './WorkOrdersTable';

interface PropertyMaintenanceSectionProps {
  propertyId: string;
}

/**
 * Property-scoped zoom level of the Maintenance feature — same table
 * and drawer as the global /maintenance page, fixed to this property,
 * no property filter/bulk actions. Mirrors UnitsSection's role for
 * Units.
 */
export function PropertyMaintenanceSection({ propertyId }: PropertyMaintenanceSectionProps) {
  const [sortKey, setSortKey] = useState<WorkOrderSortKey | undefined>(undefined);
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [addOpen, setAddOpen] = useState(false);
  const [openWorkOrderId, setOpenWorkOrderId] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<WorkOrderRow | null>(null);
  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);

  const { data, isLoading, isError, error, refetch } = useWorkOrders({
    propertyId,
    sort: sortKey,
    order: sortDir,
    limit: rowsPerPage,
    offset: page * rowsPerPage,
  });
  const deleteWorkOrder = useDeleteWorkOrder();

  const workOrders = data?.workOrders ?? [];
  const total = data?.total ?? 0;
  const emptyState: MaintenanceEmptyState = isLoading || total > 0 ? { kind: 'none' } : { kind: 'first-time' };

  function handleSort(key: WorkOrderSortKey) {
    if (sortKey === key) setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    else {
      setSortKey(key);
      setSortDir('asc');
    }
    setPage(0);
  }

  function errorMessage(err: unknown): string {
    return err instanceof Error ? err.message : 'Something went wrong. Please try again.';
  }

  return (
    <Paper variant="outlined" sx={{ p: 3 }}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 2, flexWrap: 'wrap', gap: 1 }}>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>Maintenance</Typography>
        <Stack direction="row" spacing={2.5} sx={{ alignItems: 'center' }}>
          <Link
            component={RouterLink}
            to={`/maintenance?property=${propertyId}`}
            sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, fontSize: 12.5, fontWeight: 600 }}
          >
            View in Maintenance <ArrowForwardOutlined sx={{ fontSize: 15 }} />
          </Link>
          <Button size="small" variant="contained" startIcon={<AddOutlined />} onClick={() => setAddOpen(true)}>
            Add Work Order
          </Button>
        </Stack>
      </Stack>

      {isError ? (
        <ErrorMessage error={error} onRetry={() => void refetch()} />
      ) : (
        <ErrorBoundary>
          <WorkOrdersTable
            loading={isLoading}
            workOrders={workOrders}
            total={total}
            emptyState={emptyState}
            selected={selected}
            onToggleRow={(id) =>
              setSelected((prev) => {
                const next = new Set(prev);
                if (next.has(id)) next.delete(id);
                else next.add(id);
                return next;
              })
            }
            onToggleSelectAllOnPage={() => {
              const pageIds = workOrders.map((w) => w.id);
              const allSelected = pageIds.length > 0 && pageIds.every((id) => selected.has(id));
              setSelected((prev) => {
                const next = new Set(prev);
                if (allSelected) pageIds.forEach((id) => next.delete(id));
                else pageIds.forEach((id) => next.add(id));
                return next;
              });
            }}
            sortKey={sortKey ?? ('' as WorkOrderSortKey)}
            sortDir={sortDir}
            onSort={handleSort}
            page={page}
            rowsPerPage={rowsPerPage}
            onPageChange={setPage}
            onRowsPerPageChange={(n) => {
              setRowsPerPage(n);
              setPage(0);
            }}
            onRowClick={(w) => setOpenWorkOrderId(w.id)}
            onDeleteRow={setDeleteTarget}
            onAddWorkOrder={() => setAddOpen(true)}
            onResetView={() => setPage(0)}
          />
        </ErrorBoundary>
      )}

      <WorkOrderDrawer
        open={addOpen}
        onClose={() => setAddOpen(false)}
        propertyId={propertyId}
        onSaved={() => {
          setAddOpen(false);
          setToast({ message: 'Work order created', severity: 'success' });
        }}
        onDeleted={() => setAddOpen(false)}
      />

      <WorkOrderDrawer
        open={Boolean(openWorkOrderId)}
        onClose={() => setOpenWorkOrderId(null)}
        workOrderId={openWorkOrderId ?? undefined}
        onSaved={() => setToast({ message: 'Work order updated', severity: 'success' })}
        onDeleted={() => {
          setOpenWorkOrderId(null);
          setToast({ message: 'Work order deleted', severity: 'success' });
        }}
      />

      <Dialog open={Boolean(deleteTarget)} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete this work order?</DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>
            {deleteTarget ? `"${deleteTarget.title}"` : 'This work order'} will be permanently removed. This can&apos;t be undone.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 0 }}>
          <Button variant="text" onClick={() => setDeleteTarget(null)} sx={{ color: tokens.slate[600] }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color="error"
            disabled={deleteWorkOrder.isPending}
            onClick={() => {
              if (!deleteTarget) return;
              deleteWorkOrder.mutate(deleteTarget.id, {
                onSuccess: () => {
                  setToast({ message: 'Work order deleted', severity: 'success' });
                  setDeleteTarget(null);
                },
                onError: (err) => setToast({ message: errorMessage(err), severity: 'error' }),
              });
            }}
          >
            Delete
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
    </Paper>
  );
}
