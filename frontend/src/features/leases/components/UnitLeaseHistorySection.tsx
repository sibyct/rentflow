import { useState } from 'react';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
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
import type { LeasePortfolioSortKey } from '../api/leasesApi';
import { useDeleteLease, useLeasesPortfolio } from '../hooks/useLeasesQueries';
import type { LeaseRowWithUnitProperty } from '../types';
import { LeaseFormDialog } from './LeaseFormDialog';
import { LeasesTable, type LeasesPortfolioEmptyState } from './LeasesTable';

interface UnitLeaseHistorySectionProps {
  unitId: string;
  /** Whether "Add Lease" should be offered — mirrors the unit's own status, matching UnitFormDialog's edit-mode check for an existing active lease (a unit already occupied has nothing to add here). */
  canAddLease: boolean;
}

/**
 * Unit-scoped Lease History (FR1-FR5, FR12 of the Unit Detail redesign)
 * — every lease this unit has ever had, oldest and newest included, no
 * status filter (that's what makes past/expired/terminated leases show
 * up here at all). Same table/dialogs as the global /leases page and
 * PropertyLeasesSection, fixed to this one unit.
 */
export function UnitLeaseHistorySection({ unitId, canAddLease }: UnitLeaseHistorySectionProps) {
  const navigate = useNavigate();
  const [sortKey, setSortKey] = useState<LeasePortfolioSortKey>('startDate');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);
  const [addOpen, setAddOpen] = useState(false);
  const [editLease, setEditLease] = useState<LeaseRowWithUnitProperty | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<LeaseRowWithUnitProperty | null>(null);
  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);

  const deleteLease = useDeleteLease(unitId);

  const { data, isLoading, isError, error, refetch } = useLeasesPortfolio({
    unitId,
    sort: sortKey,
    order: sortDir,
    limit: rowsPerPage,
    offset: page * rowsPerPage,
  });

  const leases = data?.leases ?? [];
  const total = data?.total ?? 0;
  // FR12 falls out of real data with no special-casing: an empty result
  // here just means no leases exist yet for this unit.
  const emptyState: LeasesPortfolioEmptyState = isLoading || total > 0 ? { kind: 'none' } : { kind: 'first-time' };

  function handleSort(key: LeasePortfolioSortKey) {
    if (sortKey === key) setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    else {
      setSortKey(key);
      setSortDir('asc');
    }
    setPage(0);
  }

  return (
    <Paper variant="outlined" sx={{ p: 3 }}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 2, flexWrap: 'wrap', gap: 1 }}>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>Lease History</Typography>
        <Link
          component={RouterLink}
          to={`/leases?unit=${unitId}`}
          sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, fontSize: 12.5, fontWeight: 600 }}
        >
          View in Leases <ArrowForwardOutlined sx={{ fontSize: 15 }} />
        </Link>
      </Stack>

      {isError ? (
        <ErrorMessage error={error} onRetry={() => void refetch()} />
      ) : total === 0 && !isLoading ? (
        <Stack spacing={1.5} sx={{ py: 3, alignItems: 'center' }}>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[400] }}>No leases yet for this unit.</Typography>
          {canAddLease && (
            <Button size="small" variant="contained" startIcon={<AddOutlined />} onClick={() => setAddOpen(true)}>
              Create Lease
            </Button>
          )}
        </Stack>
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
            onResetView={() => setPage(0)}
            onViewLease={(l) => navigate(`/leases/${l.id}`)}
            onEditLease={setEditLease}
            onDeleteLease={setDeleteTarget}
          />
        </ErrorBoundary>
      )}

      <LeaseFormDialog
        open={addOpen}
        onClose={() => setAddOpen(false)}
        unitId={unitId}
        onSaved={() => {
          setAddOpen(false);
          setToast({ message: 'Lease added', severity: 'success' });
        }}
      />

      <LeaseFormDialog
        open={Boolean(editLease)}
        onClose={() => setEditLease(null)}
        unitId={unitId}
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
            {deleteTarget ? `The lease for "${deleteTarget.primaryResidentName}"` : 'This lease'} will be permanently removed. This can&apos;t be undone.
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
    </Paper>
  );
}
