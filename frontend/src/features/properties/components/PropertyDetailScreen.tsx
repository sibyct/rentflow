import { useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import LinearProgress from '@mui/material/LinearProgress';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import AddHomeOutlined from '@mui/icons-material/AddHomeOutlined';
import ArchiveOutlined from '@mui/icons-material/ArchiveOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';
import PaymentsOutlined from '@mui/icons-material/PaymentsOutlined';
import UnarchiveOutlined from '@mui/icons-material/UnarchiveOutlined';
import type { SvgIconComponent } from '@mui/icons-material';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import { ERROR_STATE_PRESETS, ErrorState, LoadingSpinner } from '@/shared/components';
import { PropertyFinancialsSection } from '@/features/accounting';
import { PropertyLeasesSection } from '@/features/leases';
import { PropertyMaintenanceSection } from '@/features/maintenance';
import { useWorkOrders } from '@/features/maintenance/hooks/useWorkOrdersQueries';
import { UnitsSection } from '@/features/units';
import { useUnits } from '@/features/units/hooks/useUnitsQueries';
import type { UnitRow } from '@/features/units/types';
import { useProperty, useUpdatePropertyStatus } from '../hooks/usePropertiesQueries';
import type { PropertyRow, PropertyRowStatus } from '../mock/propertyRows';
import { PropertyFormModal } from './PropertyFormModal';

// A residential_single_unit property IS its one (backend-auto-created,
// not separately managed) unit — see PropertyService.CreateProperty —
// so the Units tab/section only makes sense for property types that can
// genuinely have more than one.
const SHOWS_UNITS_TAB: Record<string, boolean> = {
  'Residential – Single Unit': false,
  'Residential – Multi Unit': true,
  Commercial: true,
  'Mixed Use': true,
};

const STATUS_COLOR: Record<PropertyRowStatus, 'success' | 'info' | 'default'> = {
  Active: 'success',
  Onboarding: 'info',
  Archived: 'default',
};

// A simple, fixed presentation threshold — not a per-property
// configurable target (there's no such field on the backend) — just
// what "below target" means for the occupancy stat tile's chip/bar.
const OCCUPANCY_TARGET_PCT = 90;

const OPEN_STATUSES = new Set(['new', 'assigned', 'in_progress', 'on_hold']);

const currency = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });
const dateFormatter = new Intl.DateTimeFormat('en-US', { dateStyle: 'medium' });
const dateTimeFormatter = new Intl.DateTimeFormat('en-US', { dateStyle: 'medium', timeStyle: 'short' });

function formatDate(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : dateFormatter.format(d);
}

/** "Just now" / "12 minutes ago" / "3 hours ago" / "Yesterday" / "5 days ago", falling back to an absolute date once it's far enough back that relative time stops being useful. */
function relativeTime(iso: string): string {
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return iso;
  const diffSec = Math.round((Date.now() - then) / 1000);
  if (diffSec < 60) return 'Just now';
  const diffMin = Math.round(diffSec / 60);
  if (diffMin < 60) return `${diffMin} minute${diffMin === 1 ? '' : 's'} ago`;
  const diffHr = Math.round(diffMin / 60);
  if (diffHr < 24) return `${diffHr} hour${diffHr === 1 ? '' : 's'} ago`;
  const diffDay = Math.round(diffHr / 24);
  if (diffDay === 1) return 'Yesterday';
  if (diffDay < 14) return `${diffDay} days ago`;
  return dateFormatter.format(then);
}

/** Whole days between vacatedAt and now, for "Vacant · N days". */
function daysSince(iso: string): number {
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return 0;
  return Math.max(0, Math.floor((Date.now() - then) / 86_400_000));
}

type TabKey = 'overview' | 'units' | 'leases' | 'maintenance' | 'financials';

interface PropertyDetailScreenProps {
  propertyId: string;
}

export function PropertyDetailScreen({ propertyId }: PropertyDetailScreenProps) {
  const { data: property, isLoading, isError, error, refetch } = useProperty(propertyId);
  const updateStatus = useUpdatePropertyStatus();

  const [tab, setTab] = useState<TabKey>('overview');
  const [editOpen, setEditOpen] = useState(false);
  const [editInitialStep, setEditInitialStep] = useState(0);
  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);

  const showsUnitsTab = property ? SHOWS_UNITS_TAB[property.type] : false;

  // Backs both the Overview tab's compact Units preview and the Units
  // stat tile — already the exact query UnitsSection itself uses, so
  // TanStack Query dedupes this into one request when both are visible.
  const { data: units } = useUnits(showsUnitsTab ? propertyId : undefined);

  // A property-sized page (not the portfolio), so a generous limit
  // fetches everything there is to count rather than needing a
  // dedicated property-scoped summary endpoint.
  const { data: workOrdersData } = useWorkOrders({ propertyId, limit: 200, offset: 0 });
  const workOrders = workOrdersData?.workOrders ?? [];
  const openWorkOrders = workOrders.filter((w) => OPEN_STATUSES.has(w.status));
  const overdueWorkOrders = openWorkOrders.filter((w) => w.isOverdue);

  function openEdit(stepIndex = 0) {
    setEditInitialStep(stepIndex);
    setEditOpen(true);
  }

  if (isLoading) {
    return (
      <Box sx={{ py: 8 }}>
        <LoadingSpinner label="Loading property…" />
      </Box>
    );
  }

  if (isError) {
    if (error instanceof ApiError && error.status === 404) {
      return (
        <ErrorState
          {...ERROR_STATE_PRESETS.notFound}
          title="We can't find that property"
          message="It may have been removed, or the link is wrong. Check the address bar, or head back to your properties."
          primaryAction={{ label: 'Back to Properties', href: '/properties' }}
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

  if (!property) return null;

  const nextStatus: PropertyRowStatus = property.status === 'Archived' ? 'Active' : 'Archived';

  function toggleArchive() {
    updateStatus.mutate(
      { ids: [propertyId], status: nextStatus },
      {
        onSuccess: () =>
          setToast({ message: `${nextStatus === 'Archived' ? 'Archived' : 'Restored'} ${property!.name}`, severity: 'success' }),
        onError: (err) =>
          setToast({ message: err instanceof Error ? err.message : 'Something went wrong. Please try again.', severity: 'error' }),
      },
    );
  }

  const mapHref = `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(property.address)}`;

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
        <Typography sx={{ fontSize: 13, color: tokens.slate[400] }}>/</Typography>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>{property.name}</Typography>
      </Stack>

      <Stack direction="row" sx={{ alignItems: 'flex-start', justifyContent: 'space-between', gap: 2, flexWrap: 'wrap', mb: 1 }}>
        <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
          <Typography variant="h3">{property.name}</Typography>
          <Chip label={property.status} color={STATUS_COLOR[property.status]} size="small" sx={{ fontWeight: 600 }} />
        </Stack>
        <Stack direction="row" spacing={1.5}>
          <Button variant="outlined" startIcon={<EditOutlined />} onClick={() => openEdit(0)} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
            Edit property
          </Button>
          <Button
            variant="outlined"
            startIcon={property.status === 'Archived' ? <UnarchiveOutlined /> : <ArchiveOutlined />}
            onClick={toggleArchive}
            disabled={updateStatus.isPending}
            sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}
          >
            {property.status === 'Archived' ? 'Restore' : 'Archive'}
          </Button>
          {/* Real navigation, not a bespoke one-click flow: there's no
              single "the" lease to collect from without a specific row
              — this jumps straight to this property's own Rent Roll,
              where every outstanding balance can be recorded for real. */}
          <Button variant="contained" startIcon={<PaymentsOutlined />} onClick={() => setTab('financials')}>
            Collect rent
          </Button>
        </Stack>
      </Stack>

      <Typography sx={{ fontSize: 13.5, color: tokens.slate[500], mb: 2.5 }}>
        {property.address}{' '}
        <Link href={mapHref} target="_blank" rel="noopener noreferrer" sx={{ fontSize: 13.5 }}>
          Map
        </Link>
      </Typography>

      {property.status === 'Archived' && (
        <Alert severity="warning" icon={false} sx={{ bgcolor: tokens.warningTint, color: tokens.warningInk, mb: 2.5, '& .MuiAlert-message': { width: '100%' } }}>
          <strong>Archived.</strong> This property is no longer active — occupancy and collections aren&apos;t being tracked. Restore it to resume reporting.
        </Alert>
      )}

      <Stack direction="row" spacing={2} sx={{ flexWrap: 'wrap', mb: 2.5 }}>
        <StatTile label="Units" value={String(property.unitCount)} secondary={property.unitCount > 0 ? undefined : 'No units yet'} />
        <OccupancyStatTile occupancyPct={property.occupancyPct} tracked={property.unitCount > 0} />
        {/* collected_this_month approximates rent owed by occupied
            units (see backend/internal/domain/unit.go's
            PropertyUnitStats doc comment) — there's no payments-history
            feature yet to compute a real "vs last month" delta from,
            so this stays a plain current figure rather than inventing one. */}
        <StatTile
          label="Collected this month"
          value={property.unitCount > 0 ? currency.format(property.collectedThisMonth) : '—'}
          secondary={property.unitCount > 0 ? undefined : 'Not tracked'}
        />
        <WorkOrdersStatTile open={openWorkOrders.length} overdue={overdueWorkOrders} onViewOverdue={() => setTab('maintenance')} />
      </Stack>

      <Tabs value={tab} onChange={(_e, v: TabKey) => setTab(v)} sx={{ borderBottom: `1px solid ${tokens.slate[200]}`, mb: 2.5 }}>
        <Tab value="overview" label="Overview" sx={{ textTransform: 'none', fontWeight: 600 }} />
        {showsUnitsTab && <Tab value="units" label="Units" sx={{ textTransform: 'none', fontWeight: 600 }} />}
        <Tab value="leases" label="Leases" sx={{ textTransform: 'none', fontWeight: 600 }} />
        <Tab value="maintenance" label="Maintenance" sx={{ textTransform: 'none', fontWeight: 600 }} />
        <Tab value="financials" label="Financials" sx={{ textTransform: 'none', fontWeight: 600 }} />
      </Tabs>

      {tab === 'overview' && (
        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2.5} sx={{ alignItems: 'flex-start' }}>
          <Stack spacing={2.5} sx={{ flex: '2 1 480px', minWidth: 0, width: '100%' }}>
            <Paper variant="outlined" sx={{ p: 3 }}>
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 2 }}>Property details</Typography>
              <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(3, 1fr)' }, gap: 2 }}>
                <Field label="Property type" value={property.type} />
                <Field label="Address line 1" value={property.addressLine1} />
                <Field label="Address line 2" value={property.addressLine2} />
                <Field label="City" value={property.city} />
                <Field label="State" value={property.stateProvince} />
                <Field label="Postal code" value={property.postalCode} />
                <Field label="Country" value={property.country} />
                <Field label="Year built" value={property.yearBuilt ? String(property.yearBuilt) : ''} />
                <Field label="Onboard date" value={property.onboardDate ? formatDate(property.onboardDate) : ''} />
              </Box>

              <Box sx={{ borderTop: `1px solid ${tokens.slate[100]}`, mt: 3, pt: 2.5 }}>
                <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
                  <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>Amenities</Typography>
                </Stack>
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                  {property.amenities.map((a) => (
                    <Chip key={a} label={a} size="small" variant="outlined" />
                  ))}
                  <Chip
                    label="+ Add"
                    size="small"
                    variant="outlined"
                    onClick={() => openEdit(2)}
                    sx={{ borderStyle: 'dashed', borderColor: tokens.slate[300], color: tokens.slate[500], cursor: 'pointer' }}
                  />
                </Box>
              </Box>
            </Paper>

            {showsUnitsTab && <UnitsPreview units={units ?? []} onManage={() => setTab('units')} />}

            <Paper variant="outlined" sx={{ p: 3 }}>
              <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>Internal notes</Typography>
                <Link component="button" onClick={() => openEdit(3)} sx={{ fontSize: 12.5, fontWeight: 600 }}>
                  + Add note
                </Link>
              </Stack>
              <Typography sx={{ fontSize: 13.5, color: property.notes ? tokens.slate[700] : tokens.slate[400], whiteSpace: 'pre-wrap' }}>
                {property.notes || 'No internal notes yet.'}
              </Typography>
            </Paper>
          </Stack>

          <Stack spacing={2.5} sx={{ flex: '1 1 260px', minWidth: 0, width: '100%' }}>
            <Paper variant="outlined" sx={{ p: 3 }}>
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 2 }}>Ownership</Typography>
              <Stack spacing={1.5}>
                <OwnershipRow label="Ownership" value={property.ownership === 'managed' ? 'Managed' : property.ownership === 'owned' ? 'Owned' : '—'} />
                {property.ownership === 'managed' && <OwnershipRow label="Owner" value={property.ownerName || '—'} />}
              </Stack>
            </Paper>

            <Paper variant="outlined" sx={{ p: 3 }}>
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 1.5 }}>Quick actions</Typography>
              <Stack spacing={1}>
                {showsUnitsTab && <QuickActionRow icon={AddHomeOutlined} label="Add a unit" onClick={() => setTab('units')} />}
                <QuickActionRow icon={AddHomeOutlined} label="Add a lease" onClick={() => setTab('leases')} />
                <QuickActionRow icon={EditOutlined} label="Add a work order" onClick={() => setTab('maintenance')} />
                <QuickActionRow icon={PaymentsOutlined} label="Record a payment" onClick={() => setTab('financials')} />
              </Stack>
            </Paper>

            <Paper variant="outlined" sx={{ p: 3 }}>
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 2.5 }}>Activity</Typography>
              <ActivityTimeline
                entries={[
                  // Newest first, like any real changelog — and only
                  // ever these two entries: created_at/updated_at are
                  // the only timestamps this record keeps. A per-change
                  // history (each edit as its own entry) would need a
                  // real audit-log feature — there isn't one behind
                  // this yet.
                  ...(property.updatedAt !== property.createdAt
                    ? [
                        {
                          icon: property.status === 'Archived' ? ArchiveOutlined : EditOutlined,
                          label: property.status === 'Archived' ? 'Archived' : 'Last updated',
                          iso: property.updatedAt,
                        },
                      ]
                    : []),
                  { icon: AddHomeOutlined, label: 'Property added', iso: property.createdAt },
                ]}
              />
            </Paper>
          </Stack>
        </Stack>
      )}

      {tab === 'units' && showsUnitsTab && <UnitsSection propertyId={propertyId} />}
      {tab === 'leases' && <PropertyLeasesSection propertyId={propertyId} />}
      {tab === 'maintenance' && <PropertyMaintenanceSection propertyId={propertyId} />}
      {tab === 'financials' && <PropertyFinancialsSection property={property} />}

      <PropertyFormModal
        open={editOpen}
        onClose={() => setEditOpen(false)}
        propertyId={propertyId}
        initialStepIndex={editInitialStep}
        onSaved={(_row: PropertyRow) => {
          setEditOpen(false);
          setToast({ message: 'Property updated successfully', severity: 'success' });
        }}
        onArchiveToggled={(message) => {
          setEditOpen(false);
          setToast({ message, severity: 'success' });
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

function StatTile({ label, value, secondary, children }: { label: string; value: string; secondary?: string; children?: React.ReactNode }) {
  return (
    <Paper variant="outlined" sx={{ p: 2, flex: '1 1 180px', minWidth: 160 }}>
      <Typography sx={{ fontSize: 11, fontWeight: 600, letterSpacing: '0.04em', color: tokens.slate[500], textTransform: 'uppercase', mb: 0.5 }}>
        {label}
      </Typography>
      <Typography sx={{ fontSize: 18, fontWeight: 600, color: tokens.slate[900] }}>{value}</Typography>
      {secondary && <Typography sx={{ fontSize: 12, color: tokens.slate[400], mt: 0.25 }}>{secondary}</Typography>}
      {children}
    </Paper>
  );
}

function OccupancyStatTile({ occupancyPct, tracked }: { occupancyPct: number; tracked: boolean }) {
  const belowTarget = tracked && occupancyPct < OCCUPANCY_TARGET_PCT;
  return (
    <Paper variant="outlined" sx={{ p: 2, flex: '1 1 180px', minWidth: 160 }}>
      <Typography sx={{ fontSize: 11, fontWeight: 600, letterSpacing: '0.04em', color: tokens.slate[500], textTransform: 'uppercase', mb: 0.5 }}>
        Occupancy
      </Typography>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
        <Typography sx={{ fontSize: 18, fontWeight: 600, color: tokens.slate[900] }}>{tracked ? `${occupancyPct}%` : '—'}</Typography>
        {belowTarget && <Chip label="Below target" size="small" sx={{ bgcolor: tokens.warningTint, color: tokens.warningInk, fontWeight: 600, height: 20, fontSize: 11 }} />}
      </Stack>
      {tracked ? (
        <LinearProgress
          variant="determinate"
          value={Math.min(100, occupancyPct)}
          sx={{
            mt: 1,
            height: 5,
            borderRadius: 999,
            bgcolor: tokens.slate[100],
            '& .MuiLinearProgress-bar': { bgcolor: belowTarget ? tokens.warningInk : tokens.brand.green, borderRadius: 999 },
          }}
        />
      ) : (
        <Typography sx={{ fontSize: 12, color: tokens.slate[400], mt: 0.25 }}>Not tracked</Typography>
      )}
    </Paper>
  );
}

function WorkOrdersStatTile({ open, overdue, onViewOverdue }: { open: number; overdue: { title: string; unitName: string }[]; onViewOverdue: () => void }) {
  const firstOverdue = overdue[0];
  return (
    <Paper variant="outlined" sx={{ p: 2, flex: '1 1 180px', minWidth: 160 }}>
      <Typography sx={{ fontSize: 11, fontWeight: 600, letterSpacing: '0.04em', color: tokens.slate[500], textTransform: 'uppercase', mb: 0.5 }}>
        Open work orders
      </Typography>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
        <Typography sx={{ fontSize: 18, fontWeight: 600, color: tokens.slate[900] }}>{open}</Typography>
        {overdue.length > 0 && (
          <Chip
            label="Overdue"
            size="small"
            onClick={onViewOverdue}
            sx={{ bgcolor: 'rgba(194,38,47,0.1)', color: tokens.error, fontWeight: 600, height: 20, fontSize: 11, cursor: 'pointer' }}
          />
        )}
      </Stack>
      {firstOverdue ? (
        <Link component="button" type="button" onClick={onViewOverdue} sx={{ fontSize: 12, mt: 0.25, display: 'block', textAlign: 'left' }}>
          {firstOverdue.title} — {firstOverdue.unitName || 'Property-wide'}
        </Link>
      ) : (
        <Typography sx={{ fontSize: 12, color: tokens.slate[400], mt: 0.25 }}>{open === 0 ? 'All clear' : 'None overdue'}</Typography>
      )}
    </Paper>
  );
}

function QuickActionRow({ icon: Icon, label, onClick }: { icon: SvgIconComponent; label: string; onClick: () => void }) {
  return (
    <Link
      component="button"
      type="button"
      onClick={onClick}
      underline="none"
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        px: 1.25,
        py: 1,
        borderRadius: `${tokens.radiusControl}px`,
        bgcolor: tokens.slate[50],
        color: tokens.slate[700],
        fontSize: 13,
        fontWeight: 600,
        textAlign: 'left',
        width: '100%',
        '&:hover': { bgcolor: tokens.slate[100] },
      }}
    >
      <Icon sx={{ fontSize: 17, color: tokens.slate[500] }} />
      {label}
    </Link>
  );
}

/** Read-only, compact preview of a property's units for the Overview tab — the full add/edit/bulk-add management UI lives on the Units tab (see UnitsSection); this just orients and links there. */
function UnitsPreview({ units, onManage }: { units: UnitRow[]; onManage: () => void }) {
  const preview = units.slice(0, 4);
  return (
    <Paper variant="outlined" sx={{ p: 3 }}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: preview.length > 0 ? 1.5 : 0 }}>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>Units</Typography>
        <Link component="button" type="button" onClick={onManage} sx={{ fontSize: 12.5, fontWeight: 600 }}>
          Manage units →
        </Link>
      </Stack>
      {preview.length === 0 ? (
        <Typography sx={{ fontSize: 13, color: tokens.slate[400] }}>No units yet.</Typography>
      ) : (
        <Stack divider={<Box sx={{ borderTop: `1px solid ${tokens.slate[100]}` }} />}>
          {preview.map((u) => (
            <Stack key={u.id} direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', py: 1 }}>
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                <Box sx={{ width: 7, height: 7, borderRadius: '50%', bgcolor: u.status === 'occupied' ? tokens.brand.green : tokens.slate[300] }} />
                <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>{u.unitName}</Typography>
              </Stack>
              {u.status === 'occupied' && u.currentRent != null ? (
                <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>{currency.format(u.currentRent)}</Typography>
              ) : u.status === 'vacant' ? (
                <Typography sx={{ fontSize: 12.5, color: tokens.error }}>{u.vacatedAt ? `Vacant · ${daysSince(u.vacatedAt)} days` : 'Vacant'}</Typography>
              ) : (
                <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>—</Typography>
              )}
            </Stack>
          ))}
        </Stack>
      )}
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

function OwnershipRow({ label, value }: { label: string; value: string }) {
  return (
    <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
      <Typography sx={{ fontSize: 13, color: tokens.slate[500] }}>{label}</Typography>
      <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>{value}</Typography>
    </Stack>
  );
}

interface ActivityEntry {
  icon: SvgIconComponent;
  label: string;
  /** ISO timestamp. */
  iso: string;
}

const MARKER_SIZE = 26;

function ActivityTimeline({ entries }: { entries: ActivityEntry[] }) {
  return (
    <Stack sx={{ position: 'relative' }}>
      {entries.length > 1 && (
        <Box
          sx={{
            position: 'absolute',
            left: MARKER_SIZE / 2 - 1,
            top: MARKER_SIZE / 2,
            bottom: MARKER_SIZE / 2,
            width: '2px',
            bgcolor: tokens.slate[200],
          }}
        />
      )}
      <Stack spacing={2.5}>
        {entries.map((entry, i) => (
          <ActivityRow key={i} {...entry} />
        ))}
      </Stack>
    </Stack>
  );
}

function ActivityRow({ icon: Icon, label, iso }: ActivityEntry) {
  const absolute = (() => {
    const d = new Date(iso);
    return Number.isNaN(d.getTime()) ? iso : dateTimeFormatter.format(d);
  })();

  return (
    <Stack direction="row" spacing={1.5} sx={{ alignItems: 'flex-start' }}>
      <Box
        sx={{
          position: 'relative',
          zIndex: 1,
          width: MARKER_SIZE,
          height: MARKER_SIZE,
          borderRadius: '50%',
          bgcolor: tokens.azure[50],
          color: tokens.azure[600],
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          flexShrink: 0,
        }}
      >
        <Icon sx={{ fontSize: 15 }} />
      </Box>
      <Box sx={{ pt: 0.25 }}>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>{label}</Typography>
        <Tooltip title={absolute} placement="right">
          <Typography sx={{ fontSize: 12, color: tokens.slate[400], cursor: 'default', width: 'fit-content' }}>
            {relativeTime(iso)}
          </Typography>
        </Tooltip>
      </Box>
    </Stack>
  );
}
