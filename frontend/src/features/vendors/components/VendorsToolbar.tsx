import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import InputAdornment from '@mui/material/InputAdornment';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import AddOutlined from '@mui/icons-material/AddOutlined';
import SearchOutlined from '@mui/icons-material/SearchOutlined';
import { tokens } from '@/app/tokens';
import { WORK_ORDER_CATEGORIES, WORK_ORDER_CATEGORY_LABELS, type WorkOrderCategory } from '../types';

interface VendorsToolbarProps {
  search: string;
  onSearchChange: (value: string) => void;
  categoryFilter: WorkOrderCategory | '';
  onCategoryFilterChange: (value: WorkOrderCategory | '') => void;
  activeFilter: 'active' | 'inactive' | '';
  onActiveFilterChange: (value: 'active' | 'inactive' | '') => void;
  onAddVendor: () => void;
}

/** Filter/search row for the Vendors page — same layout pattern as MaintenanceToolbar/LeasesToolbar. */
export function VendorsToolbar({
  search,
  onSearchChange,
  categoryFilter,
  onCategoryFilterChange,
  activeFilter,
  onActiveFilterChange,
  onAddVendor,
}: VendorsToolbarProps) {
  return (
    <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center', flexWrap: 'wrap', mb: 2, rowGap: 1.25 }}>
      <TextField
        size="small"
        placeholder="Search vendors"
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
        sx={{ flex: '1 1 200px', maxWidth: 280 }}
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
        value={categoryFilter}
        onChange={(e) => onCategoryFilterChange(e.target.value as WorkOrderCategory | '')}
        sx={{ minWidth: 170 }}
      >
        <MenuItem value="">Category: All</MenuItem>
        {WORK_ORDER_CATEGORIES.map((c) => (
          <MenuItem key={c} value={c}>
            {WORK_ORDER_CATEGORY_LABELS[c]}
          </MenuItem>
        ))}
      </TextField>
      <TextField
        select
        size="small"
        slotProps={{ select: { displayEmpty: true } }}
        value={activeFilter}
        onChange={(e) => onActiveFilterChange(e.target.value as 'active' | 'inactive' | '')}
        sx={{ minWidth: 150 }}
      >
        <MenuItem value="">Status: All</MenuItem>
        <MenuItem value="active">Active</MenuItem>
        <MenuItem value="inactive">Inactive</MenuItem>
      </TextField>

      <Box sx={{ flex: 1 }} />

      <Button size="small" variant="contained" startIcon={<AddOutlined />} onClick={onAddVendor} sx={{ flexShrink: 0 }}>
        Add Vendor
      </Button>
    </Stack>
  );
}
