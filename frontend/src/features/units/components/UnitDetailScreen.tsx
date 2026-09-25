import { useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import Typography from '@mui/material/Typography';
import BuildOutlined from '@mui/icons-material/BuildOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';
import PaymentsOutlined from '@mui/icons-material/PaymentsOutlined';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import { useRecordLeasePayment, useRentRoll } from '@/features/accounting/hooks/useAccountingQueries';
import { RecordPaymentDialog } from '@/features/accounting/components/RecordPaymentDialog';
import type { RentRollRow } from '@/features/accounting/types';
import { useLeasesPortfolio } from '@/features/leases/hooks/useLeasesQueries';
import { UnitLeaseHistorySection } from '@/features/leases/components/UnitLeaseHistorySection';
import { LEASE_DISPLAY_STATUS_LABELS } from '@/features/leases/types';
import { UnitMaintenanceSection } from '@/features/maintenance/components/UnitMaintenanceSection';
import { useWorkOrders } from '@/features/maintenance/hooks/useWorkOrdersQueries';
import { useProperty } from '@/features/properties/hooks/usePropertiesQueries';
import { ERROR_STATE_PRESETS, ErrorState, LoadingSpinner } from '@/shared/components';
import { currentMonthIso, dollarsToCents, formatMoney } from '@/shared/lib/format';
import { toFormValues } from '../api/unitsApi';
import { useUnit, useUpdateUnit } from '../hooks/useUnitsQueries';
import { UnitDocumentsSection } from './UnitDocumentsSection';
import { UnitFormDialog } from './UnitFormDialog';
import { UNIT_FURNISHED_LABELS, UNIT_STATUS_LABELS, UNIT_TYPE_LABELS, type UnitStatus } from '../types';

const STATUS_COLOR: Record<UnitStatus, 'success' | 'default' | 'warning' | 'error'> = {
  occupied: 'success',
  vacant: 'default',
  maintenance: 'warning',
  off_market: 'error',
};

const OPEN_STATUSES = new Set(['new', 'assigned', 'in_progress', 'on_hold']);

const currency = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });

type TabKey = 'overview' | 'leaseHistory' | 'documents' | 'maintenance';

interface UnitDetailScreenProps {
  unitId: string;
}

function errorMessage(err: unknown): string {
  return err instanceof ApiError ? err.message : 'Something went wrong. Please try again.';
}

/**
 * Full-page Unit Detail — supersedes UnitDetailDrawer, mirroring
 * PropertyDetailScreen's tabbed shape: a unit's lease history spans
 * many leases over time (the drawer only ever showed the current one),
 * plus unit-level documents, which need more room than a drawer gives.
 */
export function UnitDetailScreen({ unitId }: UnitDetailScreenProps) {
  const { data: unit, isLoading, isError, error, refetch } = useUnit(unitId);
  const { data: property } = useProperty(unit?.propertyId);
  const updateUnit = useUpdateUnit(unit?.propertyId ?? '');

  const [tab, setTab] = useState<TabKey>('overview');
  const [editOpen, setEditOpen] = useState(false);
  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);

  // Cross-link to the Leases feature: whichever lease is currently
  // active for this unit, if any — backs both the stat tile and the
  // Collect Rent button, same lookup UnitDetailDrawer used to do.
  const { data: leaseData } = useLeasesPortfolio({ unitId, status: 'active', limit: 1, offset: 0 });
  const activeLease = leaseData?.leases[0];

  const { data: workOrdersData } = useWorkOrders({ unitId, limit: 200, offset: 0 });
  const openWorkOrders = (workOrdersData?.workOrders ?? []).filter((w) => OPEN_STATUSES.has(w.status));

  const [payTarget, setPayTarget] = useState<RentRollRow | null>(null);
  const [payError, setPayError] = useState<string | null>(null);
  const recordPayment = useRecordLeasePayment();

  // This lease's current-month rent roll row — fetched only once an
  // active lease is known, and only this one row/month rather than the
  // whole property's rent roll (see PropertyFinancialsSection, which
  // already has the full table loaded and doesn't need this).
  const month = currentMonthIso();
  const { data: rentRollData, isFetching: rentRollLoading } = useRentRoll(
    { leaseId: activeLease?.id, from: month, to: month, limit: 1, offset: 0 },
    { enabled: Boolean(activeLease) },
  );
  const rentRollRow = rentRollData?.rows[0];

  function handleCollectRent() {
    if (!rentRollRow) {
      setToast({ message: 'No outstanding rent roll row found for this lease this month.', severity: 'error' });
      return;
    }
    setPayError(null);
    setPayTarget(rentRollRow);
  }

  function markUnderMaintenance() {
    if (!unit) return;
    updateUnit.mutate(
      { id: unit.id, values: { ...toFormValues(unit), status: 'maintenance' } },
      {
        onSuccess: () => setToast({ message: 'Unit marked as under maintenance', severity: 'success' }),
        onError: (err) => setToast({ message: errorMessage(err), severity: 'error' }),
      },
    );
  }

  if (isLoading) {
    return (
      <Box sx={{ py: 8 }}>
        <LoadingSpinner label="Loading unit…" />
      </Box>
    );
  }

  if (isError) {
    if (error instanceof ApiError && error.status === 404) {
      return (
        <ErrorState
          {...ERROR_STATE_PRESETS.notFound}
          title="We can't find that unit"
          message="It may have been removed, or the link is wrong. Check the address bar, or head back to your units."
          primaryAction={{ label: 'Back to Units', href: '/units' }}
        />
      );
    }
    return (
      <ErrorState
        {...ERROR_STATE_PRESETS.serverError}
        primaryAction={{ label: 'Try again', onClick: () => void refetch() }}
      />
    );
  }

  if (!unit) return null;

  return (
    <Box>
      <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', fontSize: 13, mb: 2, flexWrap: 'wrap' }}>
        <Link component={RouterLink} to="/dashboard" sx={{ color: tokens.slate[500], textDecoration: 'none', '&:hover': { color: tokens.slate[700] } }}>
          Portfolio
        </Link>
        <Typography sx={{ fontSize: 13, color: tokens.slate[400] }}>/</Typography>
        <Link component={RouterLink} to="/properties" sx={{ color: tokens.slate[500], textDecoration: 'none', '&:hover': { color: tokens.slate[700] } }}>
          Properties
        </Link>
        {property && (
          <>
            <Typography sx={{ fontSize: 13, color: tokens.slate[400] }}>/</Typography>
            <Link
              component={RouterLink}
              to={`/properties/${property.id}`}
              sx={{ color: tokens.slate[500], textDecoration: 'none', '&:hover': { color: tokens.slate[700] } }}
            >
              {property.name}
            </Link>
          </>
        )}
        <Typography sx={{ fontSize: 13, color: tokens.slate[400] }}>/</Typography>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>{unit.unitName}</Typography>
      </Stack>

      <Stack direction="row" sx={{ alignItems: 'flex-start', justifyContent: 'space-between', gap: 2, flexWrap: 'wrap', mb: 2.5 }}>
        <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
          <Typography variant="h3">{unit.unitName}</Typography>
          <Chip label={UNIT_STATUS_LABELS[unit.status]} color={STATUS_COLOR[unit.status]} size="small" sx={{ fontWeight: 600 }} />
        </Stack>
        <Stack direction="row" spacing={1.5}>
          <Button variant="outlined" startIcon={<EditOutlined />} onClick={() => setEditOpen(true)} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
            Edit unit
          </Button>
          <Button
            variant="contained"
            startIcon={rentRollLoading ? <CircularProgress size={16} color="inherit" /> : <PaymentsOutlined />}
            onClick={handleCollectRent}
            disabled={!activeLease || rentRollLoading}
          >
            Collect rent
          </Button>
        </Stack>
      </Stack>

      <Stack direction="row" spacing={2} sx={{ flexWrap: 'wrap', mb: 2.5 }}>
        <StatTile label="Status" value={UNIT_STATUS_LABELS[unit.status]} />
        <StatTile label="Monthly rent" value={unit.currentRent != null ? currency.format(unit.currentRent) : '—'} />
        <StatTile
          label="Current lease"
          value={activeLease ? activeLease.primaryResidentName : 'None'}
          secondary={activeLease ? LEASE_DISPLAY_STATUS_LABELS[activeLease.displayStatus] : undefined}
        />
        <StatTile label="Open work orders" value={String(openWorkOrders.length)} />
      </Stack>

      <Tabs value={tab} onChange={(_e, v: TabKey) => setTab(v)} sx={{ borderBottom: `1px solid ${tokens.slate[200]}`, mb: 2.5 }}>
        <Tab value="overview" label="Overview" sx={{ textTransform: 'none', fontWeight: 600 }} />
        <Tab value="leaseHistory" label="Lease History" sx={{ textTransform: 'none', fontWeight: 600 }} />
        <Tab value="documents" label="Documents" sx={{ textTransform: 'none', fontWeight: 600 }} />
        <Tab value="maintenance" label="Maintenance" sx={{ textTransform: 'none', fontWeight: 600 }} />
      </Tabs>

      {tab === 'overview' && (
        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2.5} sx={{ alignItems: 'flex-start' }}>
          <Stack spacing={2.5} sx={{ flex: '2 1 480px', minWidth: 0, width: '100%' }}>
            <Paper variant="outlined" sx={{ p: 3 }}>
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 2 }}>Unit details</Typography>
              <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(3, 1fr)' }, gap: 2 }}>
                <Field label="Type" value={UNIT_TYPE_LABELS[unit.type] ?? unit.type} />
                <Field label="Floor" value={unit.floor} />
                <Field label="Bedrooms" value={unit.bedrooms != null ? String(unit.bedrooms) : ''} />
                <Field label="Bathrooms" value={unit.bathrooms != null ? String(unit.bathrooms) : ''} />
                <Field label="Square footage" value={unit.sqft != null ? `${unit.sqft} sq ft` : ''} />
                <Field label="Furnished" value={unit.furnished ? UNIT_FURNISHED_LABELS[unit.furnished] : ''} />
              </Box>

              <Box sx={{ borderTop: `1px solid ${tokens.slate[100]}`, mt: 3, pt: 2.5 }}>
                <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 1.5 }}>Rent</Typography>
                <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)' }, gap: 2 }}>
                  <Field label="Market rent" value={unit.marketRent != null ? currency.format(unit.marketRent) : ''} />
                  <Field label="Current rent" value={unit.currentRent != null ? currency.format(unit.currentRent) : ''} />
                  <Field label="Security deposit" value={unit.securityDeposit != null ? currency.format(unit.securityDeposit) : ''} />
                  <Field label="Rent due day" value={unit.rentDueDay != null ? String(unit.rentDueDay) : ''} />
                </Box>
              </Box>
            </Paper>

            <Paper variant="outlined" sx={{ p: 3 }}>
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 1 }}>Notes</Typography>
              <Typography sx={{ fontSize: 13.5, color: unit.notes ? tokens.slate[700] : tokens.slate[400], whiteSpace: 'pre-wrap' }}>
                {unit.notes || 'No notes yet.'}
              </Typography>
            </Paper>
          </Stack>

          <Stack spacing={2.5} sx={{ flex: '1 1 260px', minWidth: 0, width: '100%' }}>
            <Paper variant="outlined" sx={{ p: 3 }}>
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 1.5 }}>Quick actions</Typography>
              <Stack spacing={1}>
                <QuickActionRow icon={PaymentsOutlined} label="Collect rent" onClick={handleCollectRent} disabled={!activeLease || rentRollLoading} />
                <QuickActionRow
                  icon={BuildOutlined}
                  label="Mark under maintenance"
                  onClick={markUnderMaintenance}
                  disabled={unit.status === 'maintenance' || updateUnit.isPending}
                />
              </Stack>
            </Paper>
          </Stack>
        </Stack>
      )}

      {tab === 'leaseHistory' && <UnitLeaseHistorySection unitId={unitId} canAddLease={unit.status === 'vacant'} />}
      {tab === 'documents' && <UnitDocumentsSection unitId={unitId} />}
      {tab === 'maintenance' && <UnitMaintenanceSection propertyId={unit.propertyId} unitId={unitId} />}

      <UnitFormDialog
        open={editOpen}
        onClose={() => setEditOpen(false)}
        propertyId={unit.propertyId}
        unitId={unit.id}
        onSaved={() => {
          setEditOpen(false);
          setToast({ message: 'Unit updated successfully', severity: 'success' });
        }}
      />

      <RecordPaymentDialog
        open={Boolean(payTarget)}
        title="Record payment"
        subtitle={payTarget ? `${payTarget.unitName} — ${payTarget.tenantName}` : undefined}
        outstandingCents={payTarget?.outstandingCents ?? 0}
        submitting={recordPayment.isPending}
        error={payError}
        onClose={() => setPayTarget(null)}
        onSubmit={(values) => {
          if (!payTarget) return;
          recordPayment.mutate(
            { leaseId: payTarget.leaseId, values },
            {
              onSuccess: () => {
                setToast({ message: `Payment of ${formatMoney(dollarsToCents(values.amount) ?? 0)} recorded`, severity: 'success' });
                setPayTarget(null);
              },
              onError: (err) => setPayError(errorMessage(err)),
            },
          );
        }}
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

function QuickActionRow({
  icon: Icon,
  label,
  onClick,
  disabled,
}: {
  icon: typeof PaymentsOutlined;
  label: string;
  onClick: () => void;
  disabled?: boolean;
}) {
  return (
    <Link
      component="button"
      type="button"
      onClick={onClick}
      underline="none"
      disabled={disabled}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        px: 1.25,
        py: 1,
        borderRadius: `${tokens.radiusControl}px`,
        bgcolor: tokens.slate[50],
        color: disabled ? tokens.slate[400] : tokens.slate[700],
        fontSize: 13,
        fontWeight: 600,
        textAlign: 'left',
        width: '100%',
        cursor: disabled ? 'default' : 'pointer',
        '&:hover': disabled ? {} : { bgcolor: tokens.slate[100] },
      }}
    >
      <Icon sx={{ fontSize: 17, color: disabled ? tokens.slate[300] : tokens.slate[500] }} />
      {label}
    </Link>
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
