import { useState } from 'react';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import Chip from '@mui/material/Chip';
import IconButton from '@mui/material/IconButton';
import LinearProgress from '@mui/material/LinearProgress';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TablePagination from '@mui/material/TablePagination';
import TableRow from '@mui/material/TableRow';
import TableSortLabel from '@mui/material/TableSortLabel';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import AddOutlined from '@mui/icons-material/AddOutlined';
import ApartmentOutlined from '@mui/icons-material/ApartmentOutlined';
import ArchiveOutlined from '@mui/icons-material/ArchiveOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';
import FilterAltOffOutlined from '@mui/icons-material/FilterAltOffOutlined';
import MoreVertOutlined from '@mui/icons-material/MoreVertOutlined';
import SearchOffOutlined from '@mui/icons-material/SearchOffOutlined';
import UnarchiveOutlined from '@mui/icons-material/UnarchiveOutlined';
import VisibilityOutlined from '@mui/icons-material/VisibilityOutlined';
import { tokens } from '@/app/tokens';
import { EmptyState, TableSkeletonRows, type TableSkeletonColumn } from '@/shared/components';
import type { PropertySortKey } from '../api/propertiesApi';
import type { PropertyRow, PropertyRowStatus } from '../mock/propertyRows';

const STATUS_COLOR: Record<PropertyRowStatus, 'success' | 'info' | 'default'> = {
  Active: 'success',
  Onboarding: 'info',
  Archived: 'default',
};

function occupancyTier(pct: number): 'high' | 'mid' | 'low' {
  return pct >= 90 ? 'high' : pct >= 70 ? 'mid' : 'low';
}
const OCCUPANCY_COLOR: Record<'high' | 'mid' | 'low', 'success' | 'warning' | 'error'> = { high: 'success', mid: 'warning', low: 'error' };

const currency = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });

// Occupancy and collected-this-month have no sortKey: the backend has
// no leases/payments data to sort by yet (see propertiesApi.ts) — every
// property reports 0 for both — so those two columns are display-only
// until that exists.
const COLUMNS: { label: string; sortKey?: PropertySortKey; align?: 'left' | 'right' }[] = [
  { label: 'Property', sortKey: 'name' },
  { label: 'Type', sortKey: 'type' },
  { label: 'Units', sortKey: 'units', align: 'right' },
  { label: 'Occupancy', align: 'right' },
  { label: 'Collected this month', align: 'right' },
  { label: 'Status', sortKey: 'status' },
];

// Which of the (mutually exclusive) empty-state messages to show — see
// PropertiesScreen, which is the one with enough query data (an
// unfiltered portfolio count, and a count with just the status filter
// lifted) to tell these apart:
//   - 'first-time': the account has zero properties, full stop.
//   - 'status-culprit': properties exist, and lifting the status filter
//     alone would reveal them — the common "default filter is hiding
//     everything" trap.
//   - 'filtered': properties exist, but no combination this simply can
//     explain is why — search/type are involved, or lifting status
//     alone wouldn't help either.
export type PropertiesEmptyState =
  | { kind: 'none' }
  | { kind: 'first-time' }
  | { kind: 'status-culprit'; statusLabel: string; count: number }
  | { kind: 'filtered' };

const SKELETON_COLUMNS: TableSkeletonColumn[] = [
  { padding: 'checkbox', shapes: [{ variant: 'rounded', width: 18, height: 18 }] },
  { shapes: [{ width: '70%' }, { width: '45%' }] },
  { shapes: [{ width: '80%' }] },
  { align: 'right', shapes: [{ width: 24 }] },
  { align: 'right', shapes: [{ width: 40 }] },
  { align: 'right', shapes: [{ width: 60 }] },
  { shapes: [{ variant: 'rounded', width: 64, height: 22 }] },
  { padding: 'checkbox', shapes: [] },
];

interface PropertiesTableProps {
  loading: boolean;
  pageRows: PropertyRow[];
  total: number;
  emptyState: PropertiesEmptyState;
  selected: Set<string>;
  onToggleRow: (id: string) => void;
  onToggleSelectAllOnPage: () => void;
  sortKey: PropertySortKey;
  sortDir: 'asc' | 'desc';
  onSort: (key: PropertySortKey) => void;
  newRowId: string | null;
  page: number;
  rowsPerPage: number;
  onPageChange: (page: number) => void;
  onRowsPerPageChange: (rowsPerPage: number) => void;
  onRowClick: (property: PropertyRow) => void;
  onEditRow: (property: PropertyRow) => void;
  onViewDetails: (property: PropertyRow) => void;
  onToggleArchive: (property: PropertyRow) => void;
  onAddProperty: () => void;
  /** Full reset — search, type, and status all back to their defaults. */
  onResetView: () => void;
  /** Scoped reset — lifts only the status filter, leaving search/type as they were. */
  onClearStatusFilter: () => void;
}

export function PropertiesTable({
  loading,
  pageRows,
  total,
  emptyState,
  selected,
  onToggleRow,
  onToggleSelectAllOnPage,
  sortKey,
  sortDir,
  onSort,
  newRowId,
  page,
  rowsPerPage,
  onPageChange,
  onRowsPerPageChange,
  onRowClick,
  onEditRow,
  onViewDetails,
  onToggleArchive,
  onAddProperty,
  onResetView,
  onClearStatusFilter,
}: PropertiesTableProps) {
  const [kebabRow, setKebabRow] = useState<{ property: PropertyRow; anchor: HTMLElement } | null>(null);

  const pageIds = pageRows.map((p) => p.id);
  const allPageSelected = pageIds.length > 0 && pageIds.every((id) => selected.has(id));
  const somePageSelected = pageIds.some((id) => selected.has(id)) && !allPageSelected;

  return (
    <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
      <TableContainer sx={{ overflowX: 'auto' }}>
        <Table sx={{ minWidth: 860 }} size="small">
          {/* Hidden once the result set is empty and settled: a header
              row promising six columns of data above a message that
              says there isn't any reads as visually sandwiched and
              contradictory. The skeleton state keeps it, since columns
              are about to be filled in. */}
          {(loading || total > 0) && (
            <TableHead>
              <TableRow>
                <TableCell padding="checkbox">
                  <Checkbox
                    checked={allPageSelected}
                    indeterminate={somePageSelected}
                    onChange={onToggleSelectAllOnPage}
                    disabled={loading || pageRows.length === 0}
                  />
                </TableCell>
                {COLUMNS.map((col) => (
                  <TableCell key={col.label} align={col.align ?? 'left'}>
                    {col.sortKey ? (
                      <TableSortLabel
                        active={sortKey === col.sortKey}
                        direction={sortKey === col.sortKey ? sortDir : 'asc'}
                        onClick={() => onSort(col.sortKey!)}
                        // Sortable headers show their arrow at rest too
                        // (dimmed unless active), not only on hover —
                        // otherwise a sortable "Type" and a non-sortable
                        // "Occupancy" look identical until you happen to
                        // try clicking one of them.
                        sx={{ '& .MuiTableSortLabel-icon': { opacity: sortKey === col.sortKey ? 1 : 0.4 } }}
                      >
                        {col.label}
                      </TableSortLabel>
                    ) : (
                      col.label
                    )}
                  </TableCell>
                ))}
                <TableCell padding="checkbox" />
              </TableRow>
            </TableHead>
          )}
          <TableBody>
            {loading && <TableSkeletonRows columns={SKELETON_COLUMNS} />}

            {!loading && emptyState.kind === 'first-time' && (
              <TableRow>
                <TableCell colSpan={8}>
                  <EmptyState
                    icon={ApartmentOutlined}
                    title="No properties yet"
                    description="Add your first property to start tracking units, leases, and rent collection."
                    action={
                      <Button variant="contained" startIcon={<AddOutlined />} onClick={onAddProperty}>
                        Add Property
                      </Button>
                    }
                  />
                </TableCell>
              </TableRow>
            )}

            {!loading && emptyState.kind === 'status-culprit' && (
              <TableRow>
                <TableCell colSpan={8}>
                  <EmptyState
                    icon={FilterAltOffOutlined}
                    title={`Status: ${emptyState.statusLabel} is hiding ${emptyState.count} ${emptyState.count === 1 ? 'property' : 'properties'}`}
                    description={`Every property here is filtered out because none of them are currently "${emptyState.statusLabel}." Switch the status filter to see them.`}
                    action={
                      <Button variant="outlined" onClick={onClearStatusFilter} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
                        Show all statuses
                      </Button>
                    }
                  />
                </TableCell>
              </TableRow>
            )}

            {!loading && emptyState.kind === 'filtered' && (
              <TableRow>
                <TableCell colSpan={8}>
                  <EmptyState
                    icon={SearchOffOutlined}
                    title="No properties match your filters"
                    description="Try a different search term, or reset the view — search, type, and status — to see everything."
                    action={
                      <Button variant="outlined" onClick={onResetView} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
                        Reset view
                      </Button>
                    }
                  />
                </TableCell>
              </TableRow>
            )}

            {!loading &&
              pageRows.map((p) => {
                const tier = occupancyTier(p.occupancyPct);
                return (
                  <TableRow
                    key={p.id}
                    hover
                    selected={selected.has(p.id)}
                    onClick={() => onRowClick(p)}
                    sx={{
                      cursor: 'pointer',
                      ...(p.id === newRowId
                        ? { animation: 'flashNew 2.2s ease', '@keyframes flashNew': { '0%': { backgroundColor: 'rgba(15,133,119,0.14)' }, '70%': { backgroundColor: 'rgba(15,133,119,0.14)' }, '100%': { backgroundColor: 'transparent' } } }
                        : {}),
                    }}
                  >
                    <TableCell padding="checkbox" onClick={(e) => e.stopPropagation()}>
                      <Checkbox checked={selected.has(p.id)} onChange={() => onToggleRow(p.id)} />
                    </TableCell>
                    <TableCell>
                      <Typography sx={{ fontWeight: 600, fontSize: 13.5 }}>{p.name}</Typography>
                      <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>{p.address}</Typography>
                    </TableCell>
                    <TableCell>
                      <Typography sx={{ fontSize: 12.5, color: tokens.slate[600] }}>{p.type}</Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Typography sx={{ fontFamily: tokens.fontMono, fontSize: 13.5 }}>{p.units}</Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Stack sx={{ alignItems: 'flex-end', gap: 0.5 }}>
                        <Typography sx={{ fontFamily: tokens.fontMono, fontSize: 13.5 }} color={`${OCCUPANCY_COLOR[tier]}.main`}>
                          {p.occupancyPct}%
                        </Typography>
                        <LinearProgress
                          variant="determinate"
                          value={p.occupancyPct}
                          color={OCCUPANCY_COLOR[tier]}
                          sx={{ width: 64, height: 4, borderRadius: 3, bgcolor: tokens.slate[100] }}
                        />
                      </Stack>
                    </TableCell>
                    <TableCell align="right">
                      <Typography sx={{ fontFamily: tokens.fontMono, fontSize: 13.5 }}>{currency.format(p.collectedThisMonth)}</Typography>
                    </TableCell>
                    <TableCell>
                      <Chip label={p.status} color={STATUS_COLOR[p.status]} size="small" sx={{ fontWeight: 600 }} />
                    </TableCell>
                    <TableCell padding="checkbox" onClick={(e) => e.stopPropagation()}>
                      <Stack direction="row" spacing={0.25} sx={{ justifyContent: 'flex-end' }}>
                        {/* Inline quick action for the highest-frequency
                            task, so it doesn't require opening the menu
                            every time; the kebab still covers the rest. */}
                        <Tooltip title="Edit property">
                          <IconButton size="small" onClick={() => onEditRow(p)} aria-label={`Edit ${p.name}`}>
                            <EditOutlined fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="More actions">
                          <IconButton size="small" onClick={(e) => setKebabRow({ property: p, anchor: e.currentTarget })} aria-label={`More actions for ${p.name}`}>
                            <MoreVertOutlined fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </Stack>
                    </TableCell>
                  </TableRow>
                );
              })}
          </TableBody>
        </Table>
      </TableContainer>

      {!loading && total > 0 && (
        <TablePagination
          component="div"
          count={total}
          page={page}
          onPageChange={(_e, p) => onPageChange(p)}
          rowsPerPage={rowsPerPage}
          onRowsPerPageChange={(e) => onRowsPerPageChange(parseInt(e.target.value, 10))}
          rowsPerPageOptions={[5, 10, 25]}
        />
      )}

      <Menu anchorEl={kebabRow?.anchor} open={Boolean(kebabRow)} onClose={() => setKebabRow(null)}>
        <MenuItem
          onClick={() => {
            if (kebabRow) onViewDetails(kebabRow.property);
            setKebabRow(null);
          }}
        >
          <VisibilityOutlined fontSize="small" style={{ marginRight: 10 }} /> View details
        </MenuItem>
        <MenuItem
          onClick={() => {
            if (kebabRow) onEditRow(kebabRow.property);
            setKebabRow(null);
          }}
        >
          <EditOutlined fontSize="small" style={{ marginRight: 10 }} /> Edit property
        </MenuItem>
        <MenuItem
          onClick={() => {
            if (kebabRow) onToggleArchive(kebabRow.property);
            setKebabRow(null);
          }}
        >
          {kebabRow?.property.status === 'Archived' ? (
            <>
              <UnarchiveOutlined fontSize="small" style={{ marginRight: 10 }} /> Restore
            </>
          ) : (
            <>
              <ArchiveOutlined fontSize="small" style={{ marginRight: 10 }} /> Archive
            </>
          )}
        </MenuItem>
      </Menu>
    </Paper>
  );
}
