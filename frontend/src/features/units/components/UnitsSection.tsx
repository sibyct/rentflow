import { useState } from 'react';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import AddOutlined from '@mui/icons-material/AddOutlined';
import ArrowForwardOutlined from '@mui/icons-material/ArrowForwardOutlined';
import { tokens } from '@/app/tokens';
import { useUnits } from '../hooks/useUnitsQueries';
import { BulkAddUnitsDialog } from './BulkAddUnitsDialog';
import { UnitFormDialog } from './UnitFormDialog';
import { UnitsTable } from './UnitsTable';

interface UnitsSectionProps {
  propertyId: string;
}

/**
 * The Units section of a property's detail page — only rendered by the
 * caller (PropertyDetailScreen) for residential_multi_unit, commercial,
 * and mixed_use properties; a residential_single_unit property's one
 * implicit unit isn't separately managed here (see
 * PropertyService.CreateProperty). A row click navigates to that unit's
 * own full page (/units/:id) rather than opening an in-place drawer.
 */
export function UnitsSection({ propertyId }: UnitsSectionProps) {
  const navigate = useNavigate();
  const { data: units, isLoading } = useUnits(propertyId);
  const [addOpen, setAddOpen] = useState(false);
  const [bulkOpen, setBulkOpen] = useState(false);
  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);

  return (
    <Paper variant="outlined" sx={{ p: 3 }}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 2, flexWrap: 'wrap', gap: 1 }}>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>Units</Typography>
        <Stack direction="row" spacing={2.5} sx={{ alignItems: 'center' }}>
          {/* The cross-reference that makes the property-scoped section
              and the global /units page read as two zoom levels of the
              same data, not separate features — see UnitsScreen's
              ?property= deep-link handling. */}
          <Link
            component={RouterLink}
            to={`/units?property=${propertyId}`}
            sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, fontSize: 12.5, fontWeight: 600 }}
          >
            View in Units <ArrowForwardOutlined sx={{ fontSize: 15 }} />
          </Link>
          <Link component="button" type="button" onClick={() => setBulkOpen(true)} sx={{ fontSize: 12.5, fontWeight: 600 }}>
            Bulk add
          </Link>
          <Button size="small" variant="contained" startIcon={<AddOutlined />} onClick={() => setAddOpen(true)}>
            Add Unit
          </Button>
        </Stack>
      </Stack>

      <UnitsTable loading={isLoading} units={units ?? []} onAddUnit={() => setAddOpen(true)} onRowClick={(u) => navigate(`/units/${u.id}`)} />

      <UnitFormDialog
        open={addOpen}
        onClose={() => setAddOpen(false)}
        propertyId={propertyId}
        onSaved={() => {
          setAddOpen(false);
          setToast({ message: 'Unit added', severity: 'success' });
        }}
      />

      <BulkAddUnitsDialog
        open={bulkOpen}
        onClose={() => setBulkOpen(false)}
        propertyId={propertyId}
        onSaved={(count) => {
          setBulkOpen(false);
          setToast({ message: `${count} unit${count === 1 ? '' : 's'} added`, severity: 'success' });
        }}
      />

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
