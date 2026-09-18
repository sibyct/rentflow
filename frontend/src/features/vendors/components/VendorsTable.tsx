import { useState } from 'react';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import IconButton from '@mui/material/IconButton';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Rating from '@mui/material/Rating';
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
import BusinessOutlined from '@mui/icons-material/BusinessOutlined';
import DeleteOutlineOutlined from '@mui/icons-material/DeleteOutlineOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';
import MoreVertOutlined from '@mui/icons-material/MoreVertOutlined';
import SearchOffOutlined from '@mui/icons-material/SearchOffOutlined';
import VisibilityOutlined from '@mui/icons-material/VisibilityOutlined';
import { tokens } from '@/app/tokens';
import { EmptyState, TableSkeletonRows, type TableSkeletonColumn } from '@/shared/components';
import type { VendorSortKey } from '../api/vendorsApi';
import { INSURANCE_STATUS_LABELS, WORK_ORDER_CATEGORY_LABELS, type InsuranceStatus, type VendorRow } from '../types';

const INSURANCE_COLOR: Record<InsuranceStatus, 'success' | 'warning' | 'error' | 'default'> = {
  valid: 'success',
  expiring_soon: 'warning',
  expired: 'error',
  unknown: 'default',
};

const COLUMNS: { label: string; sortKey?: VendorSortKey; align?: 'left' | 'right' }[] = [
  { label: 'Vendor', sortKey: 'name' },
  { label: 'Categories' },
  { label: 'Contact' },
  { label: 'Properties served' },
  { label: 'Open jobs', align: 'right' },
  { label: 'Rating', sortKey: 'rating' },
  { label: 'Insurance' },
  { label: 'Status' },
];

const SKELETON_COLUMNS: TableSkeletonColumn[] = [
  { shapes: [{ width: '70%' }] },
  { shapes: [{ variant: 'rounded', width: 60, height: 20 }] },
  { shapes: [{ width: '65%' }] },
  { shapes: [{ width: '50%' }] },
  { align: 'right', shapes: [{ width: 40 }] },
  { shapes: [{ width: '60%' }] },
  { shapes: [{ variant: 'rounded', width: 90, height: 22 }] },
  { shapes: [{ variant: 'rounded', width: 60, height: 22 }] },
];

export type VendorsEmptyState = { kind: 'none' } | { kind: 'first-time' } | { kind: 'filtered' };

interface VendorsTableProps {
  loading: boolean;
  vendors: VendorRow[];
  total: number;
  emptyState: VendorsEmptyState;
  sortKey: VendorSortKey;
  sortDir: 'asc' | 'desc';
  onSort: (key: VendorSortKey) => void;
  page: number;
  rowsPerPage: number;
  onPageChange: (page: number) => void;
  onRowsPerPageChange: (rowsPerPage: number) => void;
  onAddVendor: () => void;
  onResetView: () => void;
  onViewVendor: (vendor: VendorRow) => void;
  onEditVendor: (vendor: VendorRow) => void;
  onDeleteVendor: (vendor: VendorRow) => void;
  onOpenJobsClick: (vendor: VendorRow) => void;
}

/** Vendors list table — same visual pattern as LeasesTable/WorkOrdersTable (columns, sort, pagination, empty states, row kebab menu). */
export function VendorsTable({
  loading,
  vendors,
  total,
  emptyState,
  sortKey,
  sortDir,
  onSort,
  page,
  rowsPerPage,
  onPageChange,
  onRowsPerPageChange,
  onAddVendor,
  onResetView,
  onViewVendor,
  onEditVendor,
  onDeleteVendor,
  onOpenJobsClick,
}: VendorsTableProps) {
  const [kebabRow, setKebabRow] = useState<{ vendor: VendorRow; anchor: HTMLElement } | null>(null);

  return (
    <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
      <TableContainer sx={{ overflowX: 'auto' }}>
        <Table sx={{ minWidth: 1080 }} size="small">
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
                <TableCell colSpan={9}>
                  <EmptyState
                    icon={BusinessOutlined}
                    title="No vendors yet"
                    description="Add a vendor to start assigning work orders to real contractors and tracking their performance."
                    action={
                      <Button variant="contained" startIcon={<AddOutlined />} onClick={onAddVendor}>
                        Add Vendor
                      </Button>
                    }
                  />
                </TableCell>
              </TableRow>
            )}

            {!loading && emptyState.kind === 'filtered' && (
              <TableRow>
                <TableCell colSpan={9}>
                  <EmptyState
                    icon={SearchOffOutlined}
                    title="No vendors match your filters"
                    description="Try a different search term, or reset the view to see every vendor."
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
              vendors.map((v) => (
                <TableRow key={v.id} hover onClick={() => onViewVendor(v)} sx={{ cursor: 'pointer' }}>
                  <TableCell>
                    <Typography sx={{ fontWeight: 600, fontSize: 13.5 }}>{v.companyName}</Typography>
                  </TableCell>
                  <TableCell>
                    <Stack direction="row" spacing={0.5} sx={{ flexWrap: 'wrap', rowGap: 0.5 }}>
                      {v.categories.slice(0, 2).map((c) => (
                        <Chip key={c} label={WORK_ORDER_CATEGORY_LABELS[c]} size="small" variant="outlined" sx={{ fontSize: 11 }} />
                      ))}
                      {v.categories.length > 2 && (
                        <Chip label={`+${v.categories.length - 2}`} size="small" variant="outlined" sx={{ fontSize: 11 }} />
                      )}
                    </Stack>
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontSize: 13, color: tokens.slate[700] }}>{v.contactPerson || '—'}</Typography>
                    <Typography sx={{ fontSize: 11.5, color: tokens.slate[500] }}>{v.phone || v.email || ''}</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontSize: 12.5, color: tokens.slate[600] }}>
                      {v.servesAllProperties ? 'All properties' : `${v.propertiesServedCount} propert${v.propertiesServedCount === 1 ? 'y' : 'ies'}`}
                    </Typography>
                  </TableCell>
                  <TableCell align="right" onClick={(e) => e.stopPropagation()}>
                    {v.openWorkOrders > 0 ? (
                      <Chip
                        label={v.openWorkOrders}
                        size="small"
                        color="info"
                        onClick={() => onOpenJobsClick(v)}
                        sx={{ fontWeight: 600, cursor: 'pointer' }}
                      />
                    ) : (
                      <Typography sx={{ fontSize: 13, color: tokens.slate[400] }}>0</Typography>
                    )}
                  </TableCell>
                  <TableCell>
                    {v.averageRating != null ? (
                      <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center' }}>
                        <Rating value={v.averageRating} precision={0.5} readOnly size="small" />
                        <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>{v.averageRating.toFixed(1)}</Typography>
                      </Stack>
                    ) : (
                      <Typography sx={{ fontSize: 12.5, color: tokens.slate[400] }}>Not yet rated</Typography>
                    )}
                  </TableCell>
                  <TableCell>
                    <Chip label={INSURANCE_STATUS_LABELS[v.insuranceStatus]} color={INSURANCE_COLOR[v.insuranceStatus]} size="small" sx={{ fontWeight: 600 }} />
                  </TableCell>
                  <TableCell>
                    <Chip label={v.active ? 'Active' : 'Inactive'} color={v.active ? 'success' : 'default'} size="small" variant={v.active ? 'filled' : 'outlined'} sx={{ fontWeight: 600 }} />
                  </TableCell>
                  <TableCell padding="checkbox" onClick={(e) => e.stopPropagation()}>
                    <Stack direction="row" spacing={0.25} sx={{ justifyContent: 'flex-end' }}>
                      <Tooltip title="Edit vendor">
                        <IconButton size="small" onClick={() => onEditVendor(v)} aria-label={`Edit ${v.companyName}`}>
                          <EditOutlined fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="More actions">
                        <IconButton size="small" onClick={(e) => setKebabRow({ vendor: v, anchor: e.currentTarget })} aria-label={`More actions for ${v.companyName}`}>
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
            if (kebabRow) onViewVendor(kebabRow.vendor);
            setKebabRow(null);
          }}
        >
          <VisibilityOutlined fontSize="small" style={{ marginRight: 10 }} /> View profile
        </MenuItem>
        <MenuItem
          onClick={() => {
            if (kebabRow) onEditVendor(kebabRow.vendor);
            setKebabRow(null);
          }}
        >
          <EditOutlined fontSize="small" style={{ marginRight: 10 }} /> Edit vendor
        </MenuItem>
        <MenuItem
          onClick={() => {
            if (kebabRow) onDeleteVendor(kebabRow.vendor);
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
