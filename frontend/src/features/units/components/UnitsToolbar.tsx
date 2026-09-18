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
import { UNIT_STATUS_LABELS, UNIT_STATUSES, UNIT_TYPE_LABELS, UNIT_TYPES, type UnitStatus, type UnitType } from '../types';

interface UnitsToolbarProps {
  search: string;
  onSearchChange: (value: string) => void;
  properties: PropertyRow[];
  propertyFilter: string;
  onPropertyFilterChange: (propertyId: string) => void;
  statusFilter: UnitStatus | '';
  onStatusFilterChange: (value: UnitStatus | '') => void;
  typeFilter: UnitType | '';
  onTypeFilterChange: (value: UnitType | '') => void;
  onAddUnit: () => void;
}

/** Filter/search row for the global Units page — same layout pattern as PropertiesToolbar, plus a Property filter that has no equivalent there (the property-scoped page needs no such filter, since it's already scoped to one). */
export function UnitsToolbar({
  search,
  onSearchChange,
  properties,
  propertyFilter,
  onPropertyFilterChange,
  statusFilter,
  onStatusFilterChange,
  typeFilter,
  onTypeFilterChange,
  onAddUnit,
}: UnitsToolbarProps) {
  return (
    <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center', flexWrap: 'wrap', mb: 2, rowGap: 1.25 }}>
      <TextField
        size="small"
        placeholder="Search units, properties, tenants"
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
        onChange={(e) => onStatusFilterChange(e.target.value as UnitStatus | '')}
        sx={{ minWidth: 150 }}
      >
        <MenuItem value="">Status: All</MenuItem>
        {UNIT_STATUSES.map((s) => (
          <MenuItem key={s} value={s}>
            {UNIT_STATUS_LABELS[s]}
          </MenuItem>
        ))}
      </TextField>
      <TextField
        select
        size="small"
        slotProps={{ select: { displayEmpty: true } }}
        value={typeFilter}
        onChange={(e) => onTypeFilterChange(e.target.value as UnitType | '')}
        sx={{ minWidth: 160 }}
      >
        <MenuItem value="">Type: All</MenuItem>
        {UNIT_TYPES.map((t) => (
          <MenuItem key={t} value={t}>
            {UNIT_TYPE_LABELS[t]}
          </MenuItem>
        ))}
      </TextField>

      <Box sx={{ flex: 1 }} />

      <Button size="small" variant="contained" startIcon={<AddOutlined />} onClick={onAddUnit} sx={{ flexShrink: 0 }}>
        Add Unit
      </Button>
    </Stack>
  );
}
