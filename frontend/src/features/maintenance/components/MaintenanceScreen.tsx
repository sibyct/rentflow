import { useState } from 'react';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import TextField from '@mui/material/TextField';
import MenuItem from '@mui/material/MenuItem';
import Typography from '@mui/material/Typography';
import { tokens } from '@/app/tokens';
import { useProperties } from '@/features/properties/hooks/usePropertiesQueries';
import { ErrorBoundary, ErrorMessage } from '@/shared/components';
import type { WorkOrderSortKey } from '../api/workOrdersApi';
import {
  useBulkReassignWorkOrders,
  useBulkUpdateWorkOrderStatus,
  useDeleteWorkOrder,
  useWorkOrderSummary,
  useWorkOrders,
} from '../hooks/useWorkOrdersQueries';
import { WORK_ORDER_STATUSES, WORK_ORDER_STATUS_LABELS, type WorkOrderCategory, type WorkOrderPriority, type WorkOrderRow, type WorkOrderStatus } from '../types';
import { MaintenanceSummaryCounters } from './MaintenanceSummaryCounters';
import { MaintenanceToolbar } from './MaintenanceToolbar';
import { RecurringRulesTab } from './RecurringRulesTab';
import { WorkOrderDrawer } from './WorkOrderDrawer';
import { WorkOrdersTable, type MaintenanceEmptyState } from './WorkOrdersTable';

type CounterKey = 'open' | 'overdue' | 'unassigned' | 'emergency';

export function MaintenanceScreen() {
  const [tab, setTab] = useState<'work-orders' | 'scheduled'>('work-orders');

  const [search, setSearch] = useState('');
  const [propertyFilter, setPropertyFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState<WorkOrderStatus | ''>('');
  const [priorityFilter, setPriorityFilter] = useState<WorkOrderPriority | ''>('');
  const [categoryFilter, setCategoryFilter] = useState<WorkOrderCategory | ''>('');
  const [activeCounter, setActiveCounter] = useState<CounterKey | null>(null);
  const [sortKey, setSortKey] = useState<WorkOrderSortKey | undefined>(undefined);
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [addOpen, setAddOpen] = useState(false);
  const [openWorkOrderId, setOpenWorkOrderId] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<WorkOrderRow | null>(null);
  const [bulkReassignOpen, setBulkReassignOpen] = useState(false);
  const [bulkReassignValue, setBulkReassignValue] = useState('');
  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);

  const { data: propertiesData } = useProperties({ limit: 100, offset: 0 });
  const properties = propertiesData?.properties ?? [];

  const { data: summary, isLoading: isSummaryLoading } = useWorkOrderSummary();

  // The Emergency counter and the Priority filter both drive the same
  // query field — clicking the counter is just a shortcut for picking
  // "Emergency" in that dropdown, so it wins over whatever the dropdown
  // is set to, same as a user changing their mind would.
  const effectivePriorityFilter = activeCounter === 'emergency' ? 'emergency' : priorityFilter;

  const { data, isLoading, isError, error, refetch } = useWorkOrders({
    search: search || undefined,
    propertyId: propertyFilter || undefined,
    status: statusFilter || undefined,
    priority: effectivePriorityFilter || undefined,
    category: categoryFilter || undefined,
    overdue: activeCounter === 'overdue' || undefined,
    unassigned: activeCounter === 'unassigned' || undefined,
    sort: sortKey,
    order: sortDir,
    limit: rowsPerPage,
    offset: page * rowsPerPage,
  });

  const deleteWorkOrder = useDeleteWorkOrder();
  const bulkUpdateStatus = useBulkUpdateWorkOrderStatus();
  const bulkReassign = useBulkReassignWorkOrders();

  const workOrders = data?.workOrders ?? [];
  const total = data?.total ?? 0;
  const isFiltered = Boolean(search || propertyFilter || statusFilter || priorityFilter || categoryFilter || activeCounter);
  const emptyState: MaintenanceEmptyState = isLoading || total > 0 ? { kind: 'none' } : isFiltered ? { kind: 'filtered' } : { kind: 'first-time' };

  function handleSort(key: WorkOrderSortKey) {
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
    setPriorityFilter('');
    setCategoryFilter('');
    setActiveCounter(null);
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
    const pageIds = workOrders.map((w) => w.id);
    const allSelected = pageIds.length > 0 && pageIds.every((id) => selected.has(id));
    setSelected((prev) => {
      const next = new Set(prev);
      if (allSelected) pageIds.forEach((id) => next.delete(id));
      else pageIds.forEach((id) => next.add(id));
      return next;
    });
  }

  function errorMessage(err: unknown): string {
    return err instanceof Error ? err.message : 'Something went wrong. Please try again.';
  }

  return (
    <Box>
      <Tabs value={tab} onChange={(_e, v) => setTab(v)} sx={{ mb: 2.5, borderBottom: `1px solid ${tokens.slate[100]}` }}>
        <Tab label="Work Orders" value="work-orders" />
        <Tab label="Scheduled" value="scheduled" />
      </Tabs>

      {tab === 'scheduled' ? (
        <RecurringRulesTab onWorkOrderGenerated={(id) => setOpenWorkOrderId(id)} />
      ) : (
        <>
          <MaintenanceSummaryCounters
            loading={isSummaryLoading}
            summary={summary}
            active={activeCounter}
            onSelect={(key) => {
              setActiveCounter(key);
              setPage(0);
            }}
          />

          <MaintenanceToolbar
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
            priorityFilter={effectivePriorityFilter}
            onPriorityFilterChange={(v) => {
              setPriorityFilter(v);
              // A manual dropdown pick takes back over from the Emergency
              // counter shortcut, so the two controls never fight.
              if (activeCounter === 'emergency') setActiveCounter(null);
              setPage(0);
            }}
            categoryFilter={categoryFilter}
            onCategoryFilterChange={(v) => {
              setCategoryFilter(v);
              setPage(0);
            }}
            onAddWorkOrder={() => setAddOpen(true)}
          />

          {selected.size > 0 && (
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
                flexWrap: 'wrap',
              }}
            >
              <Typography sx={{ fontSize: 13 }}>
                <strong>{selected.size}</strong> selected
              </Typography>
              <Stack direction="row" spacing={1} sx={{ ml: 'auto', flexWrap: 'wrap' }}>
                <TextField
                  select
                  size="small"
                  value=""
                  displayEmpty
                  onChange={(e) => {
                    const status = e.target.value as WorkOrderStatus;
                    bulkUpdateStatus.mutate(
                      { ids: Array.from(selected), status },
                      {
                        onSuccess: (n) => {
                          setSelected(new Set());
                          setToast({ message: `Updated ${n} work order${n === 1 ? '' : 's'}`, severity: 'success' });
                        },
                        onError: (err) => setToast({ message: errorMessage(err), severity: 'error' }),
                      },
                    );
                  }}
                  sx={{ minWidth: 160, bgcolor: 'rgba(255,255,255,0.08)', '& .MuiSelect-select': { color: '#fff' }, '& .MuiOutlinedInput-notchedOutline': { borderColor: 'rgba(255,255,255,0.3)' } }}
                >
                  <MenuItem value="" disabled>
                    Change status…
                  </MenuItem>
                  {WORK_ORDER_STATUSES.map((s) => (
                    <MenuItem key={s} value={s}>
                      {WORK_ORDER_STATUS_LABELS[s]}
                    </MenuItem>
                  ))}
                </TextField>
                <Button size="small" onClick={() => setBulkReassignOpen(true)} sx={{ color: '#fff', bgcolor: 'rgba(255,255,255,0.12)', '&:hover': { bgcolor: 'rgba(255,255,255,0.22)' } }}>
                  Reassign
                </Button>
                <Button size="small" onClick={() => setSelected(new Set())} sx={{ color: '#fff', textDecoration: 'underline' }}>
                  Clear
                </Button>
              </Stack>
            </Stack>
          )}

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
                onToggleRow={toggleRow}
                onToggleSelectAllOnPage={toggleSelectAllOnPage}
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
                onResetView={resetView}
              />
            </ErrorBoundary>
          )}

          <WorkOrderDrawer
            open={addOpen}
            onClose={() => setAddOpen(false)}
            properties={properties}
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
            onSaved={() => {
              setToast({ message: 'Work order updated', severity: 'success' });
            }}
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

          <Dialog open={bulkReassignOpen} onClose={() => setBulkReassignOpen(false)} maxWidth="xs" fullWidth>
            <DialogTitle>Reassign {selected.size} work order{selected.size === 1 ? '' : 's'}</DialogTitle>
            <DialogContent>
              <TextField
                autoFocus
                label="Assign to"
                fullWidth
                value={bulkReassignValue}
                onChange={(e) => setBulkReassignValue(e.target.value)}
                placeholder="Staff or vendor name"
                sx={{ mt: 1 }}
              />
            </DialogContent>
            <DialogActions sx={{ p: 2.5, pt: 0 }}>
              <Button variant="text" onClick={() => setBulkReassignOpen(false)} sx={{ color: tokens.slate[600] }}>
                Cancel
              </Button>
              <Button
                variant="contained"
                disabled={!bulkReassignValue.trim() || bulkReassign.isPending}
                onClick={() => {
                  bulkReassign.mutate(
                    { ids: Array.from(selected), assignedTo: bulkReassignValue.trim() },
                    {
                      onSuccess: (n) => {
                        setSelected(new Set());
                        setBulkReassignOpen(false);
                        setBulkReassignValue('');
                        setToast({ message: `Reassigned ${n} work order${n === 1 ? '' : 's'}`, severity: 'success' });
                      },
                      onError: (err) => setToast({ message: errorMessage(err), severity: 'error' }),
                    },
                  );
                }}
              >
                Reassign
              </Button>
            </DialogActions>
          </Dialog>
        </>
      )}

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
