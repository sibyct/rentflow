import { useState } from 'react';
import { useNavigate, Link as RouterLink } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import DeleteOutlineOutlined from '@mui/icons-material/DeleteOutlineOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import { ERROR_STATE_PRESETS, ErrorState, LoadingSpinner } from '@/shared/components';
import { useDeleteLease, useLease } from '../hooks/useLeasesQueries';
import { LEASE_DISPLAY_STATUS_LABELS, LEASE_TYPE_LABELS, RENEWAL_STATUS_LABELS, TERMINATION_REASON_LABELS, type LeaseDisplayStatus } from '../types';
import { LeaseFormDialog } from './LeaseFormDialog';

const STATUS_COLOR: Record<LeaseDisplayStatus, 'success' | 'default' | 'warning' | 'error' | 'info'> = {
  draft: 'default',
  upcoming: 'info',
  active: 'success',
  expiring_soon: 'warning',
  expired: 'error',
  terminated: 'default',
};

const currency = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });
const dateFormatter = new Intl.DateTimeFormat('en-US', { dateStyle: 'medium' });

function formatDate(iso: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : dateFormatter.format(d);
}

interface LeaseDetailScreenProps {
  leaseId: string;
}

export function LeaseDetailScreen({ leaseId }: LeaseDetailScreenProps) {
  const navigate = useNavigate();
  const { data: lease, isLoading, isError, error, refetch } = useLease(leaseId);
  const deleteLease = useDeleteLease(lease?.unitId ?? '');

  const [editOpen, setEditOpen] = useState(false);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);

  if (isLoading) {
    return (
      <Box sx={{ py: 8 }}>
        <LoadingSpinner label="Loading lease…" />
      </Box>
    );
  }

  if (isError) {
    if (error instanceof ApiError && error.status === 404) {
      return (
        <ErrorState
          {...ERROR_STATE_PRESETS.notFound}
          title="We can't find that lease"
          message="It may have been removed, or the link is wrong. Check the address bar, or head back to Leases."
          primaryAction={{ label: 'Back to Leases', href: '/leases' }}
        />
      );
    }
    return <ErrorState {...ERROR_STATE_PRESETS.serverError} primaryAction={{ label: 'Try again', onClick: () => void refetch() }} />;
  }

  if (!lease) return null;

  function handleDelete() {
    if (!lease) return;
    deleteLease.mutate(lease.id, {
      onSuccess: () => navigate('/leases'),
      onError: () => setToast({ message: 'Something went wrong. Please try again.', severity: 'error' }),
    });
  }

  return (
    <Box>
      <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', fontSize: 13, mb: 2, flexWrap: 'wrap' }}>
        <Link component={RouterLink} to="/dashboard" sx={{ color: tokens.slate[500], textDecoration: 'none', '&:hover': { color: tokens.slate[700] } }}>
          Portfolio
        </Link>
        <Typography sx={{ fontSize: 13, color: tokens.slate[400] }}>/</Typography>
        <Link component={RouterLink} to="/leases" sx={{ color: tokens.slate[500], textDecoration: 'none', '&:hover': { color: tokens.slate[700] } }}>
          Leases
        </Link>
        <Typography sx={{ fontSize: 13, color: tokens.slate[400] }}>/</Typography>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>{lease.primaryResidentName}</Typography>
      </Stack>

      <Stack direction="row" sx={{ alignItems: 'flex-start', justifyContent: 'space-between', gap: 2, flexWrap: 'wrap', mb: 1 }}>
        <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
          <Typography variant="h3">{lease.primaryResidentName}</Typography>
          <Chip label={LEASE_DISPLAY_STATUS_LABELS[lease.displayStatus]} color={STATUS_COLOR[lease.displayStatus]} size="small" sx={{ fontWeight: 600 }} />
        </Stack>
        <Stack direction="row" spacing={1.5}>
          <Button variant="outlined" startIcon={<EditOutlined />} onClick={() => setEditOpen(true)} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
            Edit
          </Button>
          <Button variant="outlined" color="error" startIcon={<DeleteOutlineOutlined />} onClick={() => setConfirmDeleteOpen(true)} sx={{ borderColor: tokens.slate[300] }}>
            Delete
          </Button>
        </Stack>
      </Stack>

      <Typography sx={{ fontSize: 13.5, color: tokens.slate[500], mb: 2.5 }}>
        <Link component={RouterLink} to={`/properties/${lease.propertyId}`}>
          {lease.propertyName}
        </Link>{' '}
        / {lease.unitName}
      </Typography>

      <Stack direction="row" spacing={2} sx={{ flexWrap: 'wrap', mb: 2.5 }}>
        <StatTile label="Monthly rent" value={currency.format(lease.monthlyRent)} />
        <StatTile label="Lease type" value={LEASE_TYPE_LABELS[lease.leaseType]} />
        <StatTile label="Term" value={formatDate(lease.startDate)} secondary={lease.endDate ? `to ${formatDate(lease.endDate)}` : 'No end date'} />
        <StatTile label="Renewal status" value={RENEWAL_STATUS_LABELS[lease.renewalStatus]} />
      </Stack>

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2.5} sx={{ alignItems: 'flex-start' }}>
        <Stack spacing={2.5} sx={{ flex: '2 1 480px', minWidth: 0, width: '100%' }}>
          <Paper variant="outlined" sx={{ p: 3 }}>
            <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 2 }}>Parties</Typography>
            <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)' }, gap: 2 }}>
              <Field label="Primary resident" value={lease.primaryResidentName} />
              <Field label="Phone" value={lease.primaryResidentPhone} />
              <Field label="Email" value={lease.primaryResidentEmail} />
              <Field label="Emergency contact" value={lease.emergencyContact} />
              <Field label="Co-residents" value={lease.coResidents.length > 0 ? lease.coResidents.join(', ') : ''} />
            </Box>

            <Box sx={{ borderTop: `1px solid ${tokens.slate[100]}`, mt: 3, pt: 2.5 }}>
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 1.5 }}>Term</Typography>
              <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)' }, gap: 2 }}>
                <Field label="Start date" value={formatDate(lease.startDate)} />
                <Field label="End date" value={formatDate(lease.endDate)} />
                <Field label="Move-in date" value={formatDate(lease.moveInDate)} />
                <Field label="Move-out date" value={formatDate(lease.moveOutDate)} />
              </Box>
            </Box>

            <Box sx={{ borderTop: `1px solid ${tokens.slate[100]}`, mt: 3, pt: 2.5 }}>
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 1.5 }}>Financials</Typography>
              <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(3, 1fr)' }, gap: 2 }}>
                <Field label="Monthly rent" value={currency.format(lease.monthlyRent)} />
                <Field label="Security deposit" value={lease.securityDeposit != null ? currency.format(lease.securityDeposit) : ''} />
                <Field label="Deposit status" value={lease.depositStatus ? lease.depositStatus.replace('_', ' ') : ''} />
                <Field label="Rent due day" value={lease.rentDueDay != null ? String(lease.rentDueDay) : ''} />
                <Field label="Late fee amount" value={lease.lateFeeAmount != null ? currency.format(lease.lateFeeAmount) : ''} />
                <Field label="Late fee grace period" value={lease.lateFeeGraceDays != null ? `${lease.lateFeeGraceDays} days` : ''} />
              </Box>
            </Box>

            <Box sx={{ borderTop: `1px solid ${tokens.slate[100]}`, mt: 3, pt: 2.5 }}>
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 1 }}>Notes</Typography>
              <Typography sx={{ fontSize: 13.5, color: lease.notes ? tokens.slate[700] : tokens.slate[400], whiteSpace: 'pre-wrap' }}>
                {lease.notes || 'No notes yet.'}
              </Typography>
            </Box>
          </Paper>
        </Stack>

        <Stack spacing={2.5} sx={{ flex: '1 1 260px', minWidth: 0, width: '100%' }}>
          <Paper variant="outlined" sx={{ p: 3 }}>
            <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 2 }}>Renewal & termination</Typography>
            <Stack spacing={1.5}>
              <DetailRow label="Renewal status" value={RENEWAL_STATUS_LABELS[lease.renewalStatus]} />
              {lease.terminationReason && <DetailRow label="Termination reason" value={TERMINATION_REASON_LABELS[lease.terminationReason]} />}
              {lease.terminationNoticeDate && <DetailRow label="Notice date" value={formatDate(lease.terminationNoticeDate)} />}
            </Stack>
          </Paper>

          <Paper variant="outlined" sx={{ p: 3 }}>
            <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 2 }}>Signature</Typography>
            <Stack spacing={1.5}>
              <DetailRow label="Signed" value={lease.signed ? 'Yes' : 'Not yet'} />
              {lease.signedDate && <DetailRow label="Signed date" value={formatDate(lease.signedDate)} />}
            </Stack>
          </Paper>
        </Stack>
      </Stack>

      <LeaseFormDialog
        open={editOpen}
        onClose={() => setEditOpen(false)}
        unitId={lease.unitId}
        leaseId={leaseId}
        onSaved={() => {
          setEditOpen(false);
          setToast({ message: 'Lease updated successfully', severity: 'success' });
        }}
      />

      <Dialog open={confirmDeleteOpen} onClose={() => setConfirmDeleteOpen(false)}>
        <DialogTitle>Delete this lease?</DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>
            The lease for &quot;{lease.primaryResidentName}&quot; will be permanently removed. This can&apos;t be undone.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 0 }}>
          <Button variant="text" onClick={() => setConfirmDeleteOpen(false)} sx={{ color: tokens.slate[600] }}>
            Cancel
          </Button>
          <Button variant="contained" color="error" onClick={handleDelete} disabled={deleteLease.isPending}>
            Delete lease
          </Button>
        </DialogActions>
      </Dialog>

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

function StatTile({ label, value, secondary }: { label: string; value: string; secondary?: string }) {
  return (
    <Paper variant="outlined" sx={{ p: 2, flex: '1 1 180px', minWidth: 160 }}>
      <Typography sx={{ fontSize: 11, fontWeight: 600, letterSpacing: '0.04em', color: tokens.slate[500], textTransform: 'uppercase', mb: 0.5 }}>
        {label}
      </Typography>
      <Typography sx={{ fontSize: 18, fontWeight: 600, color: tokens.slate[900] }}>{value}</Typography>
      {secondary && <Typography sx={{ fontSize: 12, color: tokens.slate[400], mt: 0.25 }}>{secondary}</Typography>}
    </Paper>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <Box>
      <Typography sx={{ fontSize: 11, fontWeight: 600, letterSpacing: '0.04em', color: tokens.slate[500], textTransform: 'uppercase' }}>
        {label}
      </Typography>
      <Typography sx={{ fontSize: 13.5, color: value ? tokens.slate[700] : tokens.slate[400], mt: 0.25 }}>{value || '—'}</Typography>
    </Box>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
      <Typography sx={{ fontSize: 13, color: tokens.slate[500] }}>{label}</Typography>
      <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>{value}</Typography>
    </Stack>
  );
}
