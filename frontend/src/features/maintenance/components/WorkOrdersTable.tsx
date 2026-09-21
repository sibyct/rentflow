import { Link as RouterLink } from 'react-router-dom';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import Chip from '@mui/material/Chip';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
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
import BuildOutlined from '@mui/icons-material/BuildOutlined';
import DeleteOutlineOutlined from '@mui/icons-material/DeleteOutlineOutlined';
import SearchOffOutlined from '@mui/icons-material/SearchOffOutlined';
import { tokens } from '@/app/tokens';
import { EmptyState, TableSkeletonRows, type TableSkeletonColumn } from '@/shared/components';
import type { WorkOrderSortKey } from '../api/workOrdersApi';
import { WORK_ORDER_CATEGORY_LABELS, WORK_ORDER_PRIORITY_LABELS, WORK_ORDER_STATUS_LABELS, type WorkOrderPriority, type WorkOrderRow, type WorkOrderStatus } from '../types';

const PRIORITY_COLOR: Record<WorkOrderPriority, 'default' | 'info' | 'warning' | 'error'> = {
  low: 'default',
  medium: 'info',
  high: 'warning',
  emergency: 'error',
};

const STATUS_COLOR: Record<WorkOrderStatus, 'info' | 'primary' | 'warning' | 'default' | 'success' | 'error'> = {
  new: 'info',
  assigned: 'primary',
  in_progress: 'warning',
  on_hold: 'default',
  completed: 'success',
  cancelled: 'error',
};

const dateFormatter = new Intl.DateTimeFormat('en-US', { dateStyle: 'medium' });

function formatDate(iso: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : dateFormatter.format(d);
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  return ((parts[0]?.[0] ?? '') + (parts[1]?.[0] ?? '')).toUpperCase();
}

const COLUMNS: { label: string; sortKey?: WorkOrderSortKey }[] = [
  { label: 'Title' },
  { label: 'Property / Unit' },
  { label: 'Category' },
  { label: 'Priority', sortKey: 'priority' },
  { label: 'Status', sortKey: 'status' },
  { label: 'Assigned to' },
  { label: 'Created', sortKey: 'createdAt' },
  { label: 'Due', sortKey: 'dueDate' },
];

const SKELETON_COLUMNS: TableSkeletonColumn[] = [
  { padding: 'checkbox', shapes: [{ variant: 'rounded', width: 18, height: 18 }] },
  { shapes: [{ width: '75%' }] },
  { shapes: [{ width: '65%' }, { width: '40%' }] },
  { shapes: [{ width: '60%' }] },
  { shapes: [{ variant: 'rounded', width: 70, height: 22 }] },
  { shapes: [{ variant: 'rounded', width: 80, height: 22 }] },
  { shapes: [{ variant: 'circular', width: 24, height: 24 }] },
  { shapes: [{ width: '55%' }] },
  { shapes: [{ width: '55%' }] },
  { padding: 'checkbox', shapes: [] },
];

export type MaintenanceEmptyState = { kind: 'none' } | { kind: 'first-time' } | { kind: 'filtered' };

interface WorkOrdersTableProps {
  loading: boolean;
  workOrders: WorkOrderRow[];
  total: number;
  emptyState: MaintenanceEmptyState;
  selected: Set<string>;
  onToggleRow: (id: string) => void;
  onToggleSelectAllOnPage: () => void;
  sortKey: WorkOrderSortKey;
  sortDir: 'asc' | 'desc';
  onSort: (key: WorkOrderSortKey) => void;
  page: number;
  rowsPerPage: number;
  onPageChange: (page: number) => void;
  onRowsPerPageChange: (rowsPerPage: number) => void;
  onRowClick: (workOrder: WorkOrderRow) => void;
  onDeleteRow: (workOrder: WorkOrderRow) => void;
  onAddWorkOrder: () => void;
  onResetView: () => void;
}

/** Work orders table — same visual pattern as PropertiesTable/LeasesTable (columns, sort, pagination, selection, empty states). Row-level quick actions (reassign, status change) live inside the detail drawer rather than a per-row menu — see WorkOrderDrawer, which is the one place that edits a work order — so this table's own row affordance is just Delete. */
export function WorkOrdersTable({
  loading,
  workOrders,
  total,
  emptyState,
  selected,
  onToggleRow,
  onToggleSelectAllOnPage,
  sortKey,
  sortDir,
  onSort,
  page,
  rowsPerPage,
  onPageChange,
  onRowsPerPageChange,
  onRowClick,
  onDeleteRow,
  onAddWorkOrder,
  onResetView,
}: WorkOrdersTableProps) {
  const pageIds = workOrders.map((w) => w.id);
  const allPageSelected = pageIds.length > 0 && pageIds.every((id) => selected.has(id));
  const somePageSelected = pageIds.some((id) => selected.has(id)) && !allPageSelected;

  return (
    <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
      <TableContainer sx={{ overflowX: 'auto' }}>
        <Table sx={{ minWidth: 1080 }} size="small">
          {(loading || total > 0) && (
            <TableHead>
              <TableRow>
                <TableCell padding="checkbox">
                  <Checkbox checked={allPageSelected} indeterminate={somePageSelected} onChange={onToggleSelectAllOnPage} disabled={loading || workOrders.length === 0} />
                </TableCell>
                {COLUMNS.map((col) => (
                  <TableCell key={col.label}>
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
                    icon={BuildOutlined}
                    title="No maintenance requests"
                    description="Create a work order to start tracking repairs and maintenance across your portfolio."
                    action={
                      <Button variant="contained" startIcon={<AddOutlined />} onClick={onAddWorkOrder}>
                        New Work Order
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
                    title="No work orders match your filters"
                    description="Try a different search term, or reset the view to see everything."
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
              workOrders.map((w) => (
                <TableRow
                  key={w.id}
                  hover
                  selected={selected.has(w.id)}
                  onClick={() => onRowClick(w)}
                  sx={{
                    cursor: 'pointer',
                    ...(w.priority === 'emergency' ? { borderLeft: `3px solid ${tokens.error}` } : {}),
                  }}
                >
                  <TableCell padding="checkbox" onClick={(e) => e.stopPropagation()}>
                    <Checkbox checked={selected.has(w.id)} onChange={() => onToggleRow(w.id)} />
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontWeight: 600, fontSize: 13.5 }}>{w.title}</Typography>
                  </TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Link component={RouterLink} to={`/properties/${w.propertyId}`} sx={{ fontSize: 13, fontWeight: 600, display: 'block' }}>
                      {w.propertyName}
                    </Link>
                    {w.unitName && <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>{w.unitName}</Typography>}
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontSize: 12.5, color: tokens.slate[600] }}>{WORK_ORDER_CATEGORY_LABELS[w.category]}</Typography>
                  </TableCell>
                  <TableCell>
                    <Chip label={WORK_ORDER_PRIORITY_LABELS[w.priority]} color={PRIORITY_COLOR[w.priority]} size="small" sx={{ fontWeight: 600 }} />
                  </TableCell>
                  <TableCell>
                    <Chip label={WORK_ORDER_STATUS_LABELS[w.status]} color={STATUS_COLOR[w.status]} size="small" variant="outlined" sx={{ fontWeight: 600 }} />
                  </TableCell>
                  <TableCell>
                    {w.assignedTo ? (
                      <Tooltip title={w.assignedTo}>
                        <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                          <Stack
                            sx={{
                              width: 24,
                              height: 24,
                              borderRadius: '50%',
                              bgcolor: tokens.azure[50],
                              color: tokens.azure[700],
                              alignItems: 'center',
                              justifyContent: 'center',
                              fontSize: 10,
                              fontWeight: 700,
                              flexShrink: 0,
                            }}
                          >
                            {initials(w.assignedTo)}
                          </Stack>
                          <Typography sx={{ fontSize: 12.5, color: tokens.slate[700], maxWidth: 100, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {w.assignedTo}
                          </Typography>
                        </Stack>
                      </Tooltip>
                    ) : (
                      <Typography sx={{ fontSize: 12.5, color: tokens.slate[400] }}>Unassigned</Typography>
                    )}
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontSize: 12.5, color: tokens.slate[600] }}>{formatDate(w.createdAt)}</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontSize: 12.5, fontWeight: w.isOverdue ? 700 : 400, color: w.isOverdue ? tokens.error : tokens.slate[600] }}>
                      {formatDate(w.dueDate)}
                    </Typography>
                  </TableCell>
                  <TableCell padding="checkbox" onClick={(e) => e.stopPropagation()}>
                    <Tooltip title="Delete">
                      <IconButton size="small" onClick={() => onDeleteRow(w)} aria-label={`Delete ${w.title}`}>
                        <DeleteOutlineOutlined fontSize="small" />
                      </IconButton>
                    </Tooltip>
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
    </Paper>
  );
}
