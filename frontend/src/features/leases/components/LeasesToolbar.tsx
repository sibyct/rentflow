import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import InputAdornment from '@mui/material/InputAdornment';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import AddOutlined from '@mui/icons-material/AddOutlined';
import SearchOutlined from '@mui/icons-material/SearchOutlined';
import { tokens } from '@/app/tokens';
import type { PropertyRow } from '@/features/properties/mock/propertyRows';
import { LEASE_DISPLAY_STATUS_LABELS, LEASE_DISPLAY_STATUSES, type LeaseDisplayStatus } from '../types';

interface LeasesToolbarProps {
  search: string;
  onSearchChange: (value: string) => void;
  properties: PropertyRow[];
  propertyFilter: string;
  onPropertyFilterChange: (propertyId: string) => void;
  statusFilter: LeaseDisplayStatus | '';
  onStatusFilterChange: (value: LeaseDisplayStatus | '') => void;
  onAddLease: () => void;
}

/** Filter/search row for the global Leases page — same layout pattern as UnitsToolbar/PropertiesToolbar. Status filters on the richer, date-derived display status (upcoming/expiring soon/expired/...), not the three raw stored states, since that's what a manager actually wants to slice by — e.g. everything expiring soon. */
export function LeasesToolbar({
  search,
  onSearchChange,
  properties,
  propertyFilter,
  onPropertyFilterChange,
  statusFilter,
  onStatusFilterChange,
  onAddLease,
}: LeasesToolbarProps) {
  return (
    <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center', flexWrap: 'wrap', mb: 2, rowGap: 1.25 }}>
      <TextField
        size="small"
        placeholder="Search units, properties, residents"
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
        sx={{ flex: '1 1 200px', maxWidth: 300 }}
        slotProps={{
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <SearchOutlined sx={{ fontSize: 18, color: tokens.slate[400] }} />
              </InputAdornment>
            ),
          },
        }}
      />
      <TextField
        select
        size="small"
        slotProps={{ select: { displayEmpty: true } }}
        value={propertyFilter}
        onChange={(e) => onPropertyFilterChange(e.target.value)}
        sx={{ minWidth: 190 }}
      >
        <MenuItem value="">Property: All</MenuItem>
        {properties.map((p) => (
          <MenuItem key={p.id} value={p.id}>
            {p.name}
          </MenuItem>
        ))}
      </TextField>
      <TextField
        select
        size="small"
        slotProps={{ select: { displayEmpty: true } }}
        value={statusFilter}
        onChange={(e) => onStatusFilterChange(e.target.value as LeaseDisplayStatus | '')}
        sx={{ minWidth: 170 }}
      >
        <MenuItem value="">Status: All</MenuItem>
        {LEASE_DISPLAY_STATUSES.map((s) => (
          <MenuItem key={s} value={s}>
            {LEASE_DISPLAY_STATUS_LABELS[s]}
          </MenuItem>
        ))}
      </TextField>

      <Box sx={{ flex: 1 }} />

      <Button size="small" variant="contained" startIcon={<AddOutlined />} onClick={onAddLease} sx={{ flexShrink: 0 }}>
        Add Lease
      </Button>
    </Stack>
  );
}
