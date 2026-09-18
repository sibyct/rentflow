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
import {
  WORK_ORDER_CATEGORIES,
  WORK_ORDER_CATEGORY_LABELS,
  WORK_ORDER_PRIORITIES,
  WORK_ORDER_PRIORITY_LABELS,
  WORK_ORDER_STATUSES,
  WORK_ORDER_STATUS_LABELS,
  type WorkOrderCategory,
  type WorkOrderPriority,
  type WorkOrderStatus,
} from '../types';

interface MaintenanceToolbarProps {
  search: string;
  onSearchChange: (value: string) => void;
  properties: PropertyRow[];
  propertyFilter: string;
  onPropertyFilterChange: (propertyId: string) => void;
  statusFilter: WorkOrderStatus | '';
  onStatusFilterChange: (value: WorkOrderStatus | '') => void;
  priorityFilter: WorkOrderPriority | '';
  onPriorityFilterChange: (value: WorkOrderPriority | '') => void;
  categoryFilter: WorkOrderCategory | '';
  onCategoryFilterChange: (value: WorkOrderCategory | '') => void;
  onAddWorkOrder: () => void;
}

/** Filter/search row for the Work Orders tab — same layout pattern as LeasesToolbar/UnitsToolbar. Date-range filtering isn't included yet (search + the four dropdowns cover the common cases this table needs today). */
export function MaintenanceToolbar({
  search,
  onSearchChange,
  properties,
  propertyFilter,
  onPropertyFilterChange,
  statusFilter,
  onStatusFilterChange,
  priorityFilter,
  onPriorityFilterChange,
  categoryFilter,
  onCategoryFilterChange,
  onAddWorkOrder,
}: MaintenanceToolbarProps) {
  return (
    <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center', flexWrap: 'wrap', mb: 2, rowGap: 1.25 }}>
      <TextField
        size="small"
        placeholder="Search title, description, unit"
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
      <TextField select size="small" slotProps={{ select: { displayEmpty: true } }} value={propertyFilter} onChange={(e) => onPropertyFilterChange(e.target.value)} sx={{ minWidth: 170 }}>
        <MenuItem value="">Property: All</MenuItem>
        {properties.map((p) => (
          <MenuItem key={p.id} value={p.id}>
            {p.name}
          </MenuItem>
        ))}
      </TextField>
      <TextField select size="small" slotProps={{ select: { displayEmpty: true } }} value={statusFilter} onChange={(e) => onStatusFilterChange(e.target.value as WorkOrderStatus | '')} sx={{ minWidth: 150 }}>
        <MenuItem value="">Status: All</MenuItem>
        {WORK_ORDER_STATUSES.map((s) => (
          <MenuItem key={s} value={s}>
            {WORK_ORDER_STATUS_LABELS[s]}
          </MenuItem>
        ))}
      </TextField>
      <TextField select size="small" slotProps={{ select: { displayEmpty: true } }} value={priorityFilter} onChange={(e) => onPriorityFilterChange(e.target.value as WorkOrderPriority | '')} sx={{ minWidth: 150 }}>
        <MenuItem value="">Priority: All</MenuItem>
        {WORK_ORDER_PRIORITIES.map((p) => (
          <MenuItem key={p} value={p}>
            {WORK_ORDER_PRIORITY_LABELS[p]}
          </MenuItem>
        ))}
      </TextField>
      <TextField select size="small" slotProps={{ select: { displayEmpty: true } }} value={categoryFilter} onChange={(e) => onCategoryFilterChange(e.target.value as WorkOrderCategory | '')} sx={{ minWidth: 160 }}>
        <MenuItem value="">Category: All</MenuItem>
        {WORK_ORDER_CATEGORIES.map((c) => (
          <MenuItem key={c} value={c}>
            {WORK_ORDER_CATEGORY_LABELS[c]}
          </MenuItem>
        ))}
      </TextField>

      <Box sx={{ flex: 1 }} />

      <Button size="small" variant="contained" startIcon={<AddOutlined />} onClick={onAddWorkOrder} sx={{ flexShrink: 0 }}>
        New Work Order
      </Button>
    </Stack>
  );
}
