import { useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import AddHomeOutlined from '@mui/icons-material/AddHomeOutlined';
import ArchiveOutlined from '@mui/icons-material/ArchiveOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';
import UnarchiveOutlined from '@mui/icons-material/UnarchiveOutlined';
import type { SvgIconComponent } from '@mui/icons-material';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import { ERROR_STATE_PRESETS, ErrorState, LoadingSpinner } from '@/shared/components';
import { UnitsSection } from '@/features/units';
import { useProperty, useUpdatePropertyStatus } from '../hooks/usePropertiesQueries';
import type { PropertyRow, PropertyRowStatus } from '../mock/propertyRows';
import { PropertyFormModal } from './PropertyFormModal';

// A residential_single_unit property IS its one (backend-auto-created,
// not separately managed) unit — see PropertyService.CreateProperty —
// so the Units section only makes sense for property types that can
// genuinely have more than one.
const SHOWS_UNITS_SECTION: Record<string, boolean> = {
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

/** Splits "Residential – Single Unit" into a primary "Single Unit" + secondary "Residential" line; other types have no broader category to show. */
function splitTypeLabel(type: string): { primary: string; secondary?: string } {
  const [category, specific] = type.split(' – ');
  return specific ? { primary: specific, secondary: category } : { primary: type };
}

interface PropertyDetailScreenProps {
  propertyId: string;
}

export function PropertyDetailScreen({ propertyId }: PropertyDetailScreenProps) {
  const { data: property, isLoading, isError, error, refetch } = useProperty(propertyId);
  const updateStatus = useUpdatePropertyStatus();

  const [editOpen, setEditOpen] = useState(false);
  const [editInitialStep, setEditInitialStep] = useState(0);
  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);

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

  const type = splitTypeLabel(property.type);
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
            Edit
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
        <StatTile label="Type" value={type.primary} secondary={type.secondary} />
        <StatTile label="Units" value={String(property.unitCount)} secondary={property.unitCount > 0 ? undefined : 'No units yet'} />
        {/* Computed server-side from real unit rows (see
            UnitRepository.GetPropertyUnitStats) — "—" / "Not tracked"
            only while there are no units to compute from yet, not a
            fake-looking "0%" / "$0". collected_this_month still
            approximates rent owed by occupied units, not a real
            payments feature. */}
        <StatTile label="Occupancy" value={property.unitCount > 0 ? `${property.occupancyPct}%` : '—'} secondary={property.unitCount > 0 ? undefined : 'Not tracked'} />
        <StatTile
          label="Collected this month"
          value={property.unitCount > 0 ? currency.format(property.collectedThisMonth) : '—'}
          secondary={property.unitCount > 0 ? undefined : 'Not tracked'}
        />
      </Stack>

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2.5} sx={{ alignItems: 'flex-start' }}>
        <Stack spacing={2.5} sx={{ flex: '2 1 480px', minWidth: 0, width: '100%' }}>
          <Paper variant="outlined" sx={{ p: 3 }}>
            <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 2 }}>Property details</Typography>
            <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(3, 1fr)' }, gap: 2 }}>
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

            <Box sx={{ borderTop: `1px solid ${tokens.slate[100]}`, mt: 3, pt: 2.5 }}>
              <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>Internal notes</Typography>
                <Link component="button" onClick={() => openEdit(3)} sx={{ fontSize: 12.5, fontWeight: 600 }}>
                  + Add note
                </Link>
              </Stack>
              <Typography sx={{ fontSize: 13.5, color: property.notes ? tokens.slate[700] : tokens.slate[400], whiteSpace: 'pre-wrap' }}>
                {property.notes || 'No internal notes yet.'}
              </Typography>
            </Box>
          </Paper>

          {SHOWS_UNITS_SECTION[property.type] && <UnitsSection propertyId={propertyId} propertyName={property.name} />}
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
            <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 2.5 }}>Activity</Typography>
            <ActivityTimeline
              entries={[
                // Newest first, like any real changelog — and only ever
                // these two entries: created_at/updated_at are the only
                // timestamps this record keeps. A per-change history
                // (each edit as its own entry) would need a real
                // audit-log feature — there isn't one behind this yet.
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
