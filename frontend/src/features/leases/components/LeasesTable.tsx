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
import CheckCircleOutlined from '@mui/icons-material/CheckCircleOutlined';
import DeleteOutlineOutlined from '@mui/icons-material/DeleteOutlineOutlined';
import DescriptionOutlined from '@mui/icons-material/DescriptionOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';
import MoreVertOutlined from '@mui/icons-material/MoreVertOutlined';
import RadioButtonUncheckedOutlined from '@mui/icons-material/RadioButtonUncheckedOutlined';
import SearchOffOutlined from '@mui/icons-material/SearchOffOutlined';
import VisibilityOutlined from '@mui/icons-material/VisibilityOutlined';
import { tokens } from '@/app/tokens';
import { EmptyState, TableSkeletonRows, type TableSkeletonColumn } from '@/shared/components';
import type { LeasePortfolioSortKey } from '../api/leasesApi';
import { LEASE_DISPLAY_STATUS_LABELS, LEASE_TYPE_LABELS, type LeaseDisplayStatus, type LeaseRowWithUnitProperty } from '../types';

const STATUS_COLOR: Record<LeaseDisplayStatus, 'success' | 'default' | 'warning' | 'error' | 'info'> = {
  draft: 'default',
  upcoming: 'info',
  active: 'success',
  expiring_soon: 'warning',
  expired: 'error',
  terminated: 'default',
};

const currency = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });
const dateFormatter = new Intl.DateTimeFormat('en-US', { dateStyle: 'medium' });

function formatDate(iso: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : dateFormatter.format(d);
}

const COLUMNS: { label: string; sortKey?: LeasePortfolioSortKey; align?: 'left' | 'right' }[] = [
  { label: 'Property' },
  { label: 'Unit' },
  { label: 'Resident' },
  { label: 'Start', sortKey: 'startDate' },
  { label: 'End', sortKey: 'endDate' },
  { label: 'Type' },
  { label: 'Status', sortKey: 'status' },
  { label: 'Rent', sortKey: 'monthlyRent', align: 'right' },
  { label: 'Signed' },
];

const SKELETON_COLUMNS: TableSkeletonColumn[] = [
  { shapes: [{ width: '70%' }] },
  { shapes: [{ width: '60%' }] },
  { shapes: [{ width: '65%' }] },
  { shapes: [{ width: '55%' }] },
  { shapes: [{ width: '55%' }] },
  { shapes: [{ width: '55%' }] },
  { shapes: [{ variant: 'rounded', width: 72, height: 22 }] },
  { align: 'right', shapes: [{ width: 60 }] },
  { shapes: [{ variant: 'circular', width: 18, height: 18 }] },
];

export type LeasesPortfolioEmptyState = { kind: 'none' } | { kind: 'first-time' } | { kind: 'filtered' };

interface LeasesTableProps {
  loading: boolean;
  leases: LeaseRowWithUnitProperty[];
  total: number;
  emptyState: LeasesPortfolioEmptyState;
  sortKey: LeasePortfolioSortKey;
  sortDir: 'asc' | 'desc';
  onSort: (key: LeasePortfolioSortKey) => void;
  page: number;
  rowsPerPage: number;
  onPageChange: (page: number) => void;
  onRowsPerPageChange: (rowsPerPage: number) => void;
  onAddLease: () => void;
  onResetView: () => void;
  onViewLease: (lease: LeaseRowWithUnitProperty) => void;
  onEditLease: (lease: LeaseRowWithUnitProperty) => void;
  onDeleteLease: (lease: LeaseRowWithUnitProperty) => void;
}

/** Portfolio-wide leases table for the global /leases page — same visual pattern as GlobalUnitsTable/PropertiesTable (columns, sort, pagination, empty states, row kebab menu). */
export function LeasesTable({
  loading,
  leases,
  total,
  emptyState,
  sortKey,
  sortDir,
  onSort,
  page,
  rowsPerPage,
  onPageChange,
  onRowsPerPageChange,
  onAddLease,
  onResetView,
  onViewLease,
  onEditLease,
  onDeleteLease,
}: LeasesTableProps) {
  const [kebabRow, setKebabRow] = useState<{ lease: LeaseRowWithUnitProperty; anchor: HTMLElement } | null>(null);

  return (
    <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
      <TableContainer sx={{ overflowX: 'auto' }}>
        <Table sx={{ minWidth: 1040 }} size="small">
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
                <TableCell colSpan={10}>
                  <EmptyState
                    icon={DescriptionOutlined}
                    title="No leases yet"
                    description="Add a lease to any unit to start tracking rent, renewals, and term dates."
                    action={
                      <Button variant="contained" startIcon={<AddOutlined />} onClick={onAddLease}>
                        Add Lease
                      </Button>
                    }
                  />
                </TableCell>
              </TableRow>
            )}

            {!loading && emptyState.kind === 'filtered' && (
              <TableRow>
                <TableCell colSpan={10}>
                  <EmptyState
                    icon={SearchOffOutlined}
                    title="No leases match your filters"
                    description="Try a different search term, or reset the view to see every lease in your portfolio."
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
              leases.map((l) => (
                <TableRow key={l.id} hover onClick={() => onViewLease(l)} sx={{ cursor: 'pointer' }}>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Link component={RouterLink} to={`/properties/${l.propertyId}`} sx={{ fontSize: 13, fontWeight: 600 }}>
                      {l.propertyName}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontSize: 13, color: tokens.slate[700] }}>{l.unitName}</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontWeight: 600, fontSize: 13.5 }}>{l.primaryResidentName}</Typography>
                    {l.coResidents.length > 0 && (
                      <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>+{l.coResidents.length} co-resident{l.coResidents.length === 1 ? '' : 's'}</Typography>
                    )}
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontSize: 12.5, color: tokens.slate[600] }}>{formatDate(l.startDate)}</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontSize: 12.5, color: tokens.slate[600] }}>{l.endDate ? formatDate(l.endDate) : 'No end date'}</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontSize: 12.5, color: tokens.slate[600] }}>{LEASE_TYPE_LABELS[l.leaseType]}</Typography>
                  </TableCell>
                  <TableCell>
                    <Chip label={LEASE_DISPLAY_STATUS_LABELS[l.displayStatus]} color={STATUS_COLOR[l.displayStatus]} size="small" sx={{ fontWeight: 600 }} />
                  </TableCell>
                  <TableCell align="right">
                    <Typography sx={{ fontFamily: tokens.fontMono, fontSize: 13 }}>{currency.format(l.monthlyRent)}</Typography>
                  </TableCell>
                  <TableCell>
                    <Tooltip title={l.signed ? 'Signed' : 'Not yet signed'}>
                      {l.signed ? (
                        <CheckCircleOutlined sx={{ fontSize: 18, color: 'success.main' }} />
                      ) : (
                        <RadioButtonUncheckedOutlined sx={{ fontSize: 18, color: tokens.slate[300] }} />
                      )}
                    </Tooltip>
                  </TableCell>
                  <TableCell padding="checkbox" onClick={(e) => e.stopPropagation()}>
                    <Stack direction="row" spacing={0.25} sx={{ justifyContent: 'flex-end' }}>
                      <Tooltip title="Edit lease">
                        <IconButton size="small" onClick={() => onEditLease(l)} aria-label={`Edit lease for ${l.primaryResidentName}`}>
                          <EditOutlined fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="More actions">
                        <IconButton size="small" onClick={(e) => setKebabRow({ lease: l, anchor: e.currentTarget })} aria-label={`More actions for ${l.primaryResidentName}'s lease`}>
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
            if (kebabRow) onViewLease(kebabRow.lease);
            setKebabRow(null);
          }}
        >
          <VisibilityOutlined fontSize="small" style={{ marginRight: 10 }} /> View details
        </MenuItem>
        <MenuItem
          onClick={() => {
            if (kebabRow) onEditLease(kebabRow.lease);
            setKebabRow(null);
          }}
        >
          <EditOutlined fontSize="small" style={{ marginRight: 10 }} /> Edit lease
        </MenuItem>
        <MenuItem
          onClick={() => {
            if (kebabRow) onDeleteLease(kebabRow.lease);
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
