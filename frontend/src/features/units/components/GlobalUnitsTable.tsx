import { useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
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
import DeleteOutlineOutlined from '@mui/icons-material/DeleteOutlineOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';
import MeetingRoomOutlined from '@mui/icons-material/MeetingRoomOutlined';
import MoreVertOutlined from '@mui/icons-material/MoreVertOutlined';
import SearchOffOutlined from '@mui/icons-material/SearchOffOutlined';
import VisibilityOutlined from '@mui/icons-material/VisibilityOutlined';
import { tokens } from '@/app/tokens';
import { EmptyState, TableSkeletonRows, type TableSkeletonColumn } from '@/shared/components';
import type { UnitPortfolioSortKey } from '../api/unitsApi';
import { UNIT_STATUS_LABELS, UNIT_TYPE_LABELS, type UnitRowWithProperty, type UnitStatus } from '../types';

const STATUS_COLOR: Record<UnitStatus, 'success' | 'default' | 'warning' | 'error'> = {
  occupied: 'success',
  vacant: 'default',
  maintenance: 'warning',
  off_market: 'error',
};

const currency = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });

const COLUMNS: { label: string; sortKey?: UnitPortfolioSortKey; align?: 'left' | 'right' }[] = [
  { label: 'Property', sortKey: 'propertyName' },
  { label: 'Unit' },
  { label: 'Type' },
  { label: 'Beds/Baths' },
  { label: 'Status', sortKey: 'status' },
  { label: 'Rent', sortKey: 'rent', align: 'right' },
  { label: 'Tenant' },
];

const SKELETON_COLUMNS: TableSkeletonColumn[] = [
  { shapes: [{ width: '70%' }] },
  { shapes: [{ width: '60%' }, { width: '40%' }] },
  { shapes: [{ width: '65%' }] },
  { shapes: [{ width: '50%' }] },
  { shapes: [{ variant: 'rounded', width: 64, height: 22 }] },
  { align: 'right', shapes: [{ width: 60 }] },
  { shapes: [{ width: '55%' }] },
];

export type UnitsPortfolioEmptyState = { kind: 'none' } | { kind: 'first-time' } | { kind: 'filtered' };

interface GlobalUnitsTableProps {
  loading: boolean;
  units: UnitRowWithProperty[];
  total: number;
  emptyState: UnitsPortfolioEmptyState;
  sortKey: UnitPortfolioSortKey;
  sortDir: 'asc' | 'desc';
  onSort: (key: UnitPortfolioSortKey) => void;
  page: number;
  rowsPerPage: number;
  onPageChange: (page: number) => void;
  onRowsPerPageChange: (rowsPerPage: number) => void;
  onAddUnit: () => void;
  onResetView: () => void;
  onViewUnit: (unit: UnitRowWithProperty) => void;
  onEditUnit: (unit: UnitRowWithProperty) => void;
  onDeleteUnit: (unit: UnitRowWithProperty) => void;
}

/** Portfolio-wide units table for the global /units page — same visual pattern as PropertiesTable (columns, sort, pagination, empty states, row kebab menu), so the two read as siblings rather than different design languages. */
export function GlobalUnitsTable({
  loading,
  units,
  total,
  emptyState,
  sortKey,
  sortDir,
  onSort,
  page,
  rowsPerPage,
  onPageChange,
  onRowsPerPageChange,
  onAddUnit,
  onResetView,
  onViewUnit,
  onEditUnit,
  onDeleteUnit,
}: GlobalUnitsTableProps) {
  const [kebabRow, setKebabRow] = useState<{ unit: UnitRowWithProperty; anchor: HTMLElement } | null>(null);

  return (
    <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
      <TableContainer sx={{ overflowX: 'auto' }}>
        <Table sx={{ minWidth: 900 }} size="small">
          {(loading || total > 0) && (
            <TableHead>
              <TableRow>
                {COLUMNS.map((col) => (
                  <TableCell key={col.label} align={col.align ?? 'left'}>
                    {col.sortKey ? (
                      <TableSortLabel
                        active={sortKey === col.sortKey}
                        direction={sortKey === col.sortKey ? sortDir : 'asc'}
                        onClick={() => onSort(col.sortKey!)}
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
                    icon={MeetingRoomOutlined}
                    title="No units yet across your portfolio"
                    description="Add a unit to any property to start tracking occupancy and rent portfolio-wide."
                    action={
                      <Button variant="contained" startIcon={<AddOutlined />} onClick={onAddUnit}>
                        Add Unit
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
                    title="No units match your filters"
                    description="Try a different search term, or reset the view to see every unit in your portfolio."
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
              units.map((u) => (
                <TableRow key={u.id} hover onClick={() => onViewUnit(u)} sx={{ cursor: 'pointer' }}>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Link component={RouterLink} to={`/properties/${u.propertyId}`} sx={{ fontSize: 13, fontWeight: 600 }}>
                      {u.propertyName}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontWeight: 600, fontSize: 13.5 }}>{u.unitName}</Typography>
                    {u.floor && <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>Floor {u.floor}</Typography>}
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontSize: 12.5, color: tokens.slate[600] }}>{UNIT_TYPE_LABELS[u.type] ?? u.type}</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontFamily: tokens.fontMono, fontSize: 13 }}>
                      {u.bedrooms ?? '—'} bd / {u.bathrooms ?? '—'} ba
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip label={UNIT_STATUS_LABELS[u.status]} color={STATUS_COLOR[u.status]} size="small" sx={{ fontWeight: 600 }} />
                  </TableCell>
                  <TableCell align="right">
                    <Typography sx={{ fontFamily: tokens.fontMono, fontSize: 13 }}>
                      {u.currentRent != null ? currency.format(u.currentRent) : u.marketRent != null ? `${currency.format(u.marketRent)} mkt` : '—'}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontSize: 13, color: u.tenantName ? tokens.slate[700] : tokens.slate[400] }}>
                      {u.tenantName || '—'}
                    </Typography>
                  </TableCell>
                  <TableCell padding="checkbox" onClick={(e) => e.stopPropagation()}>
                    <Stack direction="row" spacing={0.25} sx={{ justifyContent: 'flex-end' }}>
                      <Tooltip title="Edit unit">
                        <IconButton size="small" onClick={() => onEditUnit(u)} aria-label={`Edit ${u.unitName}`}>
                          <EditOutlined fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="More actions">
                        <IconButton size="small" onClick={(e) => setKebabRow({ unit: u, anchor: e.currentTarget })} aria-label={`More actions for ${u.unitName}`}>
                          <MoreVertOutlined fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    </Stack>
                  </TableCell>
                </TableRow>
              ))}
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
          rowsPerPageOptions={[10, 25, 50]}
        />
      )}

      <Menu anchorEl={kebabRow?.anchor} open={Boolean(kebabRow)} onClose={() => setKebabRow(null)}>
        <MenuItem
          onClick={() => {
            if (kebabRow) onViewUnit(kebabRow.unit);
            setKebabRow(null);
          }}
        >
          <VisibilityOutlined fontSize="small" style={{ marginRight: 10 }} /> View details
        </MenuItem>
        <MenuItem
          onClick={() => {
            if (kebabRow) onEditUnit(kebabRow.unit);
            setKebabRow(null);
          }}
        >
          <EditOutlined fontSize="small" style={{ marginRight: 10 }} /> Edit unit
        </MenuItem>
        <MenuItem
          onClick={() => {
            if (kebabRow) onDeleteUnit(kebabRow.unit);
            setKebabRow(null);
          }}
          sx={{ color: 'error.main' }}
        >
          <DeleteOutlineOutlined fontSize="small" style={{ marginRight: 10 }} /> Delete
        </MenuItem>
      </Menu>
    </Paper>
  );
}
