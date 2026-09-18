import { useState } from 'react';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import IconButton from '@mui/material/IconButton';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import AddOutlined from '@mui/icons-material/AddOutlined';
import DeleteOutlineOutlined from '@mui/icons-material/DeleteOutlineOutlined';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import type { BulkUnitRow } from '../api/unitsApi';
import { useCreateUnitsBulk } from '../hooks/useUnitsQueries';
import { UNIT_TYPE_LABELS, UNIT_TYPES } from '../types';

interface BulkAddUnitsDialogProps {
  open: boolean;
  onClose: () => void;
  propertyId: string;
  onSaved: (count: number) => void;
}

function emptyRow(): BulkUnitRow {
  return { unitName: '', type: 'studio', bedrooms: '', bathrooms: '', marketRent: '' };
}

const DEFAULT_ROW_COUNT = 4;

/**
 * Spreadsheet-style bulk unit creation — for initial building setup,
 * where filling in a dozen single-unit drawers one at a time (see
 * UnitFormDrawer) would be needless friction. Rows with no name are
 * silently skipped on submit rather than validated, so a manager can
 * bump the row count up before knowing every unit's name yet.
 */
export function BulkAddUnitsDialog({ open, onClose, propertyId, onSaved }: BulkAddUnitsDialogProps) {
  const [rows, setRows] = useState<BulkUnitRow[]>(() => Array.from({ length: DEFAULT_ROW_COUNT }, emptyRow));
  const [error, setError] = useState<string | null>(null);
  const createBulk = useCreateUnitsBulk(propertyId);

  function resetAndClose() {
    setRows(Array.from({ length: DEFAULT_ROW_COUNT }, emptyRow));
    setError(null);
    onClose();
  }

  function updateRow(index: number, patch: Partial<BulkUnitRow>) {
    setRows((prev) => prev.map((r, i) => (i === index ? { ...r, ...patch } : r)));
  }

  function addRow() {
    setRows((prev) => [...prev, emptyRow()]);
  }

  function removeRow(index: number) {
    setRows((prev) => prev.filter((_, i) => i !== index));
  }

  function handleSubmit() {
    setError(null);
    const validRows = rows.filter((r) => r.unitName.trim() !== '');
    if (validRows.length === 0) {
      setError('Enter a name for at least one unit.');
      return;
    }
    createBulk.mutate(validRows, {
      onSuccess: (created) => {
        onSaved(created.length);
        resetAndClose();
      },
      onError: (err) => setError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.'),
    });
  }

  return (
    <Dialog open={open} onClose={resetAndClose} maxWidth="md" fullWidth>
      <DialogTitle>Bulk add units</DialogTitle>
      <DialogContent>
        <Typography sx={{ fontSize: 13, color: tokens.slate[500], mb: 2 }}>
          Quickly set up several units at once. You can fill in the rest of each unit&apos;s details later.
        </Typography>

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        <Stack spacing={1} sx={{ overflowX: 'auto' }}>
          <Stack direction="row" spacing={1} sx={{ px: 1, minWidth: 600 }}>
            <Typography sx={{ flex: '2 1 140px', fontSize: 11, fontWeight: 600, color: tokens.slate[500], textTransform: 'uppercase' }}>Unit name</Typography>
            <Typography sx={{ flex: '2 1 140px', fontSize: 11, fontWeight: 600, color: tokens.slate[500], textTransform: 'uppercase' }}>Type</Typography>
            <Typography sx={{ flex: '1 1 80px', fontSize: 11, fontWeight: 600, color: tokens.slate[500], textTransform: 'uppercase' }}>Beds</Typography>
            <Typography sx={{ flex: '1 1 80px', fontSize: 11, fontWeight: 600, color: tokens.slate[500], textTransform: 'uppercase' }}>Baths</Typography>
            <Typography sx={{ flex: '1 1 120px', fontSize: 11, fontWeight: 600, color: tokens.slate[500], textTransform: 'uppercase' }}>Market rent</Typography>
            <Box sx={{ width: 36 }} />
          </Stack>

          {rows.map((row, i) => (
            <Stack key={i} direction="row" spacing={1} sx={{ alignItems: 'center', minWidth: 600 }}>
              <TextField
                value={row.unitName}
                onChange={(e) => updateRow(i, { unitName: e.target.value })}
                placeholder={`Unit ${i + 1}`}
                size="small"
                sx={{ flex: '2 1 140px' }}
              />
              <TextField
                select
                value={row.type}
                onChange={(e) => updateRow(i, { type: e.target.value as BulkUnitRow['type'] })}
                size="small"
                sx={{ flex: '2 1 140px' }}
              >
                {UNIT_TYPES.map((t) => (
                  <MenuItem key={t} value={t}>
                    {UNIT_TYPE_LABELS[t]}
                  </MenuItem>
                ))}
              </TextField>
              <TextField value={row.bedrooms} onChange={(e) => updateRow(i, { bedrooms: e.target.value })} type="number" size="small" sx={{ flex: '1 1 80px' }} />
              <TextField value={row.bathrooms} onChange={(e) => updateRow(i, { bathrooms: e.target.value })} type="number" size="small" sx={{ flex: '1 1 80px' }} />
              <TextField
                value={row.marketRent}
                onChange={(e) => updateRow(i, { marketRent: e.target.value })}
                type="number"
                size="small"
                sx={{ flex: '1 1 120px' }}
                slotProps={{ input: { startAdornment: '$' } }}
              />
              <IconButton size="small" onClick={() => removeRow(i)} disabled={rows.length === 1} aria-label={`Remove row ${i + 1}`}>
                <DeleteOutlineOutlined fontSize="small" />
              </IconButton>
            </Stack>
          ))}
        </Stack>

        <Button size="small" startIcon={<AddOutlined />} onClick={addRow} sx={{ mt: 1.5 }}>
          Add row
        </Button>
      </DialogContent>
      <DialogActions sx={{ p: 2.5, pt: 0 }}>
        <Button variant="text" onClick={resetAndClose} sx={{ color: tokens.slate[600] }}>
          Cancel
        </Button>
        <Button variant="contained" onClick={handleSubmit} disabled={createBulk.isPending}>
          Add units
        </Button>
      </DialogActions>
    </Dialog>
  );
}
