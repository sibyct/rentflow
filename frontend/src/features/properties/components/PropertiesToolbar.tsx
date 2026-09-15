import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import InputAdornment from '@mui/material/InputAdornment';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import AddOutlined from '@mui/icons-material/AddOutlined';
import FileDownloadOutlined from '@mui/icons-material/FileDownloadOutlined';
import SearchOutlined from '@mui/icons-material/SearchOutlined';
import { tokens } from '@/app/tokens';
import { PROPERTY_TYPES, type PropertyRowStatus, type PropertyType } from '../mock/propertyRows';

interface PropertiesToolbarProps {
  search: string;
  onSearchChange: (value: string) => void;
  typeFilter: PropertyType | '';
  onTypeFilterChange: (value: PropertyType | '') => void;
  statusFilter: PropertyRowStatus | '';
  onStatusFilterChange: (value: PropertyRowStatus | '') => void;
  onExport: () => void;
  onAddProperty: () => void;
}

/** One toolbar row — search, filters, and the primary actions. */
export function PropertiesToolbar({
  search,
  onSearchChange,
  typeFilter,
  onTypeFilterChange,
  statusFilter,
  onStatusFilterChange,
  onExport,
  onAddProperty,
}: PropertiesToolbarProps) {
  return (
    <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center', flexWrap: 'wrap', mb: 2, rowGap: 1.25 }}>
      <TextField
        size="small"
        placeholder="Search properties, addresses"
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
      {/* displayEmpty matters here, not just as a nicety: MUI's Select
          renders nothing at all for value="" unless this is set, even
          though the "Type: All" MenuItem exists — that's what made
          these two read as blank/unlabeled next to Status (which
          defaults to "Active", a non-empty value, so it never hit this). */}
      <TextField
        select
        size="small"
        slotProps={{ select: { displayEmpty: true } }}
        value={typeFilter}
        onChange={(e) => onTypeFilterChange(e.target.value as PropertyType | '')}
        sx={{ minWidth: 170 }}
      >
        <MenuItem value="">Type: All</MenuItem>
        {PROPERTY_TYPES.map((t) => (
          <MenuItem key={t} value={t}>
            {t}
          </MenuItem>
        ))}
      </TextField>
      {/* No Occupancy filter here: occupancy_pct is a backend placeholder
          (always 0 — see propertiesApi.ts) until leases/payments exist to
          compute it from, so a filter on it would only ever return
          "everything" or "nothing" and would be actively misleading. */}
      <TextField
        select
        size="small"
        slotProps={{ select: { displayEmpty: true } }}
        value={statusFilter}
        onChange={(e) => onStatusFilterChange(e.target.value as PropertyRowStatus | '')}
        sx={{ minWidth: 140 }}
      >
        <MenuItem value="">Status: All</MenuItem>
        <MenuItem value="Active">Active</MenuItem>
        <MenuItem value="Onboarding">Onboarding</MenuItem>
        <MenuItem value="Archived">Archived</MenuItem>
      </TextField>

      <Box sx={{ flex: 1 }} />

      {/* Grouped so Export + Add Property wrap to the next line
          together on a narrow viewport, rather than splitting
          independently. */}
      <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center', flexShrink: 0 }}>
        <Button
          size="small"
          variant="outlined"
          startIcon={<FileDownloadOutlined />}
          onClick={onExport}
          sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}
        >
          Export
        </Button>
        <Button size="small" variant="contained" startIcon={<AddOutlined />} onClick={onAddProperty}>
          Add Property
        </Button>
      </Stack>
    </Stack>
  );
}
