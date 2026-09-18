import { useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import Divider from '@mui/material/Divider';
import Drawer from '@mui/material/Drawer';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import ArrowForwardOutlined from '@mui/icons-material/ArrowForwardOutlined';
import CloseOutlined from '@mui/icons-material/CloseOutlined';
import DeleteOutlineOutlined from '@mui/icons-material/DeleteOutlineOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';
import { tokens } from '@/app/tokens';
import { useLeasesPortfolio } from '@/features/leases/hooks/useLeasesQueries';
import { LEASE_DISPLAY_STATUS_LABELS } from '@/features/leases/types';
import { useDeleteUnit, useUnit } from '../hooks/useUnitsQueries';
import { UNIT_FURNISHED_LABELS, UNIT_STATUS_LABELS, UNIT_TYPE_LABELS, type UnitStatus } from '../types';

const STATUS_COLOR: Record<UnitStatus, 'success' | 'default' | 'warning' | 'error'> = {
  occupied: 'success',
  vacant: 'default',
  maintenance: 'warning',
  off_market: 'error',
};

const currency = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });

interface UnitDetailDrawerProps {
  unitId: string | null;
  propertyId: string;
  propertyName: string;
  onClose: () => void;
  onEdit: (unitId: string) => void;
  onDeleted: () => void;
}

/** Read-only unit view, opened from a UnitsTable row click — mirrors the properties feature's own view-then-explicit-edit pattern, condensed into a drawer since a unit is a secondary entity within its property's page rather than a page of its own. */
export function UnitDetailDrawer({ unitId, propertyId, propertyName, onClose, onEdit, onDeleted }: UnitDetailDrawerProps) {
  const { data: unit, isLoading } = useUnit(unitId ?? undefined);
  const deleteUnit = useDeleteUnit(propertyId);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);

  // Cross-links to the Leases feature: whichever lease is currently
  // active for this unit, if any — see LeasesSection's own "View in
  // Units" link for the reverse direction.
  const { data: leaseData } = useLeasesPortfolio(
    { unitId: unitId ?? undefined, status: 'active', limit: 1, offset: 0 },
    { enabled: Boolean(unitId) },
  );
  const activeLease = leaseData?.leases[0];

  function handleDelete() {
    if (!unitId) return;
    deleteUnit.mutate(unitId, {
      onSuccess: () => {
        setConfirmDeleteOpen(false);
        onDeleted();
      },
    });
  }

  return (
    <>
      <Drawer anchor="right" open={Boolean(unitId)} onClose={onClose} slotProps={{ paper: { sx: { width: { xs: '100%', sm: 440 } } } }}>
        <Stack sx={{ height: '100%' }}>
          <Stack sx={{ p: 2.5, borderBottom: `1px solid ${tokens.slate[100]}` }}>
            <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
              <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>
                {propertyName} / <Box component="span" sx={{ fontWeight: 600, color: tokens.slate[700] }}>{unit?.unitName ?? '…'}</Box>
              </Typography>
              <IconButton onClick={onClose} aria-label="Close" size="small">
                <CloseOutlined fontSize="small" />
              </IconButton>
            </Stack>
          </Stack>

          {isLoading || !unit ? (
            <Box sx={{ p: 3 }}>
              <Typography sx={{ fontSize: 13.5, color: tokens.slate[500] }}>Loading unit…</Typography>
            </Box>
          ) : (
            <>
              <Stack spacing={3} sx={{ p: 2.5, overflowY: 'auto', flex: 1 }}>
                <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center' }}>
                  <Typography variant="h5" sx={{ fontSize: 18 }}>{unit.unitName}</Typography>
                  <Chip label={UNIT_STATUS_LABELS[unit.status]} color={STATUS_COLOR[unit.status]} size="small" sx={{ fontWeight: 600 }} />
                </Stack>

                <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 2 }}>
                  <Field label="Type" value={UNIT_TYPE_LABELS[unit.type] ?? unit.type} />
                  <Field label="Floor" value={unit.floor} />
                  <Field label="Bedrooms" value={unit.bedrooms != null ? String(unit.bedrooms) : ''} />
                  <Field label="Bathrooms" value={unit.bathrooms != null ? String(unit.bathrooms) : ''} />
                  <Field label="Square footage" value={unit.sqft != null ? `${unit.sqft} sq ft` : ''} />
                  <Field label="Furnished" value={unit.furnished ? UNIT_FURNISHED_LABELS[unit.furnished] : ''} />
                </Box>

                <Box sx={{ borderTop: `1px solid ${tokens.slate[100]}`, pt: 2.5 }}>
                  <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 1.5 }}>Rent</Typography>
                  <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 2 }}>
                    <Field label="Market rent" value={unit.marketRent != null ? currency.format(unit.marketRent) : ''} />
                    <Field label="Current rent" value={unit.currentRent != null ? currency.format(unit.currentRent) : ''} />
                    <Field label="Security deposit" value={unit.securityDeposit != null ? currency.format(unit.securityDeposit) : ''} />
                    <Field label="Rent due day" value={unit.rentDueDay != null ? String(unit.rentDueDay) : ''} />
                  </Box>
                </Box>

                {unit.status === 'occupied' && (
                  <Box sx={{ borderTop: `1px solid ${tokens.slate[100]}`, pt: 2.5 }}>
                    <Field label="Tenant" value={unit.tenantName} />
                  </Box>
                )}

                <Box sx={{ borderTop: `1px solid ${tokens.slate[100]}`, pt: 2.5 }}>
                  <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 1 }}>Lease</Typography>
                  {activeLease ? (
                    <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
                      <Box>
                        <Typography sx={{ fontSize: 13.5, color: tokens.slate[700] }}>{activeLease.primaryResidentName}</Typography>
                        <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>{LEASE_DISPLAY_STATUS_LABELS[activeLease.displayStatus]}</Typography>
                      </Box>
                      <Link
                        component={RouterLink}
                        to={`/leases/${activeLease.id}`}
                        sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, fontSize: 12.5, fontWeight: 600 }}
                      >
                        View lease <ArrowForwardOutlined sx={{ fontSize: 15 }} />
                      </Link>
                    </Stack>
                  ) : (
                    <Typography sx={{ fontSize: 13.5, color: tokens.slate[400] }}>No active lease on file.</Typography>
                  )}
                </Box>

                <Box sx={{ borderTop: `1px solid ${tokens.slate[100]}`, pt: 2.5 }}>
                  <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 1 }}>Notes</Typography>
                  <Typography sx={{ fontSize: 13.5, color: unit.notes ? tokens.slate[700] : tokens.slate[400], whiteSpace: 'pre-wrap' }}>
                    {unit.notes || 'No notes yet.'}
                  </Typography>
                </Box>
              </Stack>

              <Divider />
              <Stack direction="row" spacing={1.5} sx={{ p: 2.5, justifyContent: 'space-between' }}>
                <Button
                  variant="text"
                  color="error"
                  startIcon={<DeleteOutlineOutlined />}
                  onClick={() => setConfirmDeleteOpen(true)}
                >
                  Delete
                </Button>
                <Button variant="contained" startIcon={<EditOutlined />} onClick={() => onEdit(unit.id)}>
                  Edit
                </Button>
              </Stack>
            </>
          )}
        </Stack>
      </Drawer>

      <Dialog open={confirmDeleteOpen} onClose={() => setConfirmDeleteOpen(false)}>
        <DialogTitle>Delete this unit?</DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>
            {unit?.unitName ? `"${unit.unitName}"` : 'This unit'} will be permanently removed. This can&apos;t be undone.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 0 }}>
          <Button variant="text" onClick={() => setConfirmDeleteOpen(false)} sx={{ color: tokens.slate[600] }}>
            Cancel
          </Button>
          <Button variant="contained" color="error" onClick={handleDelete} disabled={deleteUnit.isPending}>
            Delete unit
          </Button>
        </DialogActions>
      </Dialog>
    </>
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
