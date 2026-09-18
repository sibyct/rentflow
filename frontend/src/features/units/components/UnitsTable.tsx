import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import AddOutlined from '@mui/icons-material/AddOutlined';
import MeetingRoomOutlined from '@mui/icons-material/MeetingRoomOutlined';
import { tokens } from '@/app/tokens';
import { EmptyState, TableSkeletonRows, type TableSkeletonColumn } from '@/shared/components';
import { UNIT_STATUS_LABELS, UNIT_TYPE_LABELS, type UnitRow, type UnitStatus } from '../types';

const STATUS_COLOR: Record<UnitStatus, 'success' | 'default' | 'warning' | 'error'> = {
  occupied: 'success',
  vacant: 'default',
  maintenance: 'warning',
  off_market: 'error',
};

const currency = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });

const SKELETON_COLUMNS: TableSkeletonColumn[] = [
  { shapes: [{ width: '70%' }, { width: '40%' }] },
  { shapes: [{ width: '70%' }] },
  { shapes: [{ width: '50%' }] },
  { shapes: [{ variant: 'rounded', width: 64, height: 22 }] },
  { align: 'right', shapes: [{ width: 60 }] },
  { shapes: [{ width: '55%' }] },
];

interface UnitsTableProps {
  loading: boolean;
  units: UnitRow[];
  onAddUnit: () => void;
  onRowClick: (unit: UnitRow) => void;
}

/** Read-only units table for a property's Units section — no selection, sort, or pagination: a building's unit count rarely needs any of that (see PropertiesTable for the fuller pattern this deliberately doesn't repeat here). */
export function UnitsTable({ loading, units, onAddUnit, onRowClick }: UnitsTableProps) {
  return (
    <TableContainer sx={{ border: `1px solid ${tokens.slate[100]}`, borderRadius: `${tokens.radiusCard}px` }}>
      <Table size="small">
        {(loading || units.length > 0) && (
          <TableHead>
            <TableRow>
              <TableCell>Unit</TableCell>
              <TableCell>Type</TableCell>
              <TableCell>Beds/Baths</TableCell>
              <TableCell>Status</TableCell>
              <TableCell align="right">Rent</TableCell>
              <TableCell>Tenant</TableCell>
            </TableRow>
          </TableHead>
        )}
        <TableBody>
          {loading && <TableSkeletonRows columns={SKELETON_COLUMNS} />}

          {!loading && units.length === 0 && (
            <TableRow>
              <TableCell colSpan={6}>
                <EmptyState
                  icon={MeetingRoomOutlined}
                  title="No units yet"
                  description="Add this property's units to start tracking occupancy and rent."
                  action={
                    <Button variant="contained" startIcon={<AddOutlined />} onClick={onAddUnit}>
                      Add Unit
                    </Button>
                  }
                />
              </TableCell>
            </TableRow>
          )}

          {!loading &&
            units.map((u) => (
              <TableRow key={u.id} hover onClick={() => onRowClick(u)} sx={{ cursor: 'pointer' }}>
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
                  <Chip label={UNIT_STATUS_LABELS[u.status] ?? u.status} color={STATUS_COLOR[u.status]} size="small" sx={{ fontWeight: 600 }} />
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
              </TableRow>
            ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}
