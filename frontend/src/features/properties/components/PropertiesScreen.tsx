import { useEffect, useMemo, useState } from 'react';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import IconButton from '@mui/material/IconButton';
import InputAdornment from '@mui/material/InputAdornment';
import LinearProgress from '@mui/material/LinearProgress';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Skeleton from '@mui/material/Skeleton';
import Snackbar from '@mui/material/Snackbar';
import Alert from '@mui/material/Alert';
import Stack from '@mui/material/Stack';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TablePagination from '@mui/material/TablePagination';
import TableRow from '@mui/material/TableRow';
import TableSortLabel from '@mui/material/TableSortLabel';
import TextField from '@mui/material/TextField';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import AddOutlined from '@mui/icons-material/AddOutlined';
import ApartmentOutlined from '@mui/icons-material/ApartmentOutlined';
import ArchiveOutlined from '@mui/icons-material/ArchiveOutlined';
import DensityMediumOutlined from '@mui/icons-material/DensityMediumOutlined';
import DensitySmallOutlined from '@mui/icons-material/DensitySmallOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';
import FileDownloadOutlined from '@mui/icons-material/FileDownloadOutlined';
import MoreVertOutlined from '@mui/icons-material/MoreVertOutlined';
import SearchOffOutlined from '@mui/icons-material/SearchOffOutlined';
import SearchOutlined from '@mui/icons-material/SearchOutlined';
import UnarchiveOutlined from '@mui/icons-material/UnarchiveOutlined';
import VisibilityOutlined from '@mui/icons-material/VisibilityOutlined';
import { tokens } from '@/app/tokens';
import { AddPropertyDrawer } from './AddPropertyDrawer';
import { MOCK_PROPERTIES, PROPERTY_TYPES, type PropertyRow, type PropertyRowStatus, type PropertyType } from '../mock/propertyRows';

type SortKey = 'name' | 'type' | 'units' | 'occupancyPct' | 'collectedThisMonth' | 'status';
type OccupancyFilter = '' | 'high' | 'mid' | 'low';

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

export function PropertiesScreen() {
  const [properties, setProperties] = useState<PropertyRow[]>(MOCK_PROPERTIES);
  const [loading, setLoading] = useState(true);

  const [search, setSearch] = useState('');
  const [typeFilter, setTypeFilter] = useState<PropertyType | ''>('');
  const [occupancyFilter, setOccupancyFilter] = useState<OccupancyFilter>('');
  const [statusFilter, setStatusFilter] = useState<PropertyRowStatus | ''>('Active');

  const [sortKey, setSortKey] = useState<SortKey>('name');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [density, setDensity] = useState<'comfortable' | 'compact'>('comfortable');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [addUnitsOpen, setAddUnitsOpen] = useState(false);
  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'info' } | null>(null);
  const [newRowId, setNewRowId] = useState<string | null>(null);
  const [kebabRow, setKebabRow] = useState<{ id: string; anchor: HTMLElement } | null>(null);

  // Simulated fetch latency so the 5-skeleton-row loading state is real,
  // not just a demo toggle — mirrors what a real API call would look like
  // once this feature reads from the backend instead of mock data.
  useEffect(() => {
    const t = setTimeout(() => setLoading(false), 650);
    return () => clearTimeout(t);
  }, []);

  const filtered = useMemo(() => {
    let rows = properties.filter((p) => {
      if (statusFilter && p.status !== statusFilter) return false;
      if (typeFilter && p.type !== typeFilter) return false;
      if (occupancyFilter && occupancyTier(p.occupancyPct) !== occupancyFilter) return false;
      if (search) {
        const q = search.toLowerCase();
        if (!p.name.toLowerCase().includes(q) && !p.address.toLowerCase().includes(q)) return false;
      }
      return true;
    });
    rows = rows.slice().sort((a, b) => {
      const dir = sortDir === 'asc' ? 1 : -1;
      const av = a[sortKey];
      const bv = b[sortKey];
      if (typeof av === 'string') return av.localeCompare(bv as string) * dir;
      return ((av as number) - (bv as number)) * dir;
    });
    return rows;
  }, [properties, statusFilter, typeFilter, occupancyFilter, search, sortKey, sortDir]);

  const pageRows = filtered.slice(page * rowsPerPage, page * rowsPerPage + rowsPerPage);
  const pageIds = pageRows.map((p) => p.id);
  const allPageSelected = pageIds.length > 0 && pageIds.every((id) => selected.has(id));
  const somePageSelected = pageIds.some((id) => selected.has(id)) && !allPageSelected;

  const counts = useMemo(() => {
    const active = properties.filter((p) => p.status === 'Active').length;
    const onboarding = properties.filter((p) => p.status === 'Onboarding').length;
    const archived = properties.filter((p) => p.status === 'Archived').length;
    return { active, onboarding, archived };
  }, [properties]);
  const subtext = [
    counts.active && `${counts.active} active`,
    counts.onboarding && `${counts.onboarding} onboarding`,
    counts.archived && `${counts.archived} archived`,
  ]
    .filter(Boolean)
    .join(' · ') || '0 properties';

  function handleSort(key: SortKey) {
    if (sortKey === key) setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    else {
      setSortKey(key);
      setSortDir('asc');
    }
  }

  function toggleRow(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }
  function toggleSelectAllOnPage() {
    setSelected((prev) => {
      const next = new Set(prev);
      if (allPageSelected) pageIds.forEach((id) => next.delete(id));
      else pageIds.forEach((id) => next.add(id));
      return next;
    });
  }

  function clearFilters() {
    setSearch('');
    setTypeFilter('');
    setOccupancyFilter('');
    setStatusFilter('');
    setPage(0);
  }

  function bulkArchive() {
    const n = selected.size;
    setProperties((prev) => prev.map((p) => (selected.has(p.id) ? { ...p, status: 'Archived' } : p)));
    setSelected(new Set());
    setToast({ message: `Archived ${n} ${n === 1 ? 'property' : 'properties'}`, severity: 'success' });
  }
  function bulkExport() {
    setToast({ message: `Exported ${selected.size} ${selected.size === 1 ? 'property' : 'properties'}`, severity: 'success' });
  }

  function handleSaved(row: PropertyRow) {
    setProperties((prev) => [...prev, row]);
    setStatusFilter(''); // reveal the new Onboarding row immediately
    setPage(0);
    setNewRowId(row.id);
    setDrawerOpen(false);
    setToast({ message: 'Property added successfully', severity: 'success' });
    window.setTimeout(() => setAddUnitsOpen(true), 260);
    window.setTimeout(() => setNewRowId(null), 2400);
  }

  return (
    <Box>
      <Stack direction="row" sx={{ alignItems: 'flex-start', justifyContent: 'space-between', gap: 2, flexWrap: 'wrap', mb: 3 }}>
        <Typography sx={{ fontSize: 13.5, color: tokens.slate[500] }}>{subtext}</Typography>
        <Stack direction="row" spacing={1.5}>
          <Button
            variant="outlined"
            startIcon={<FileDownloadOutlined />}
            onClick={() => setToast({ message: 'Exporting properties as CSV…', severity: 'info' })}
            sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}
          >
            Export
          </Button>
          <Button variant="contained" startIcon={<AddOutlined />} onClick={() => setDrawerOpen(true)}>
            Add Property
          </Button>
        </Stack>
      </Stack>

      <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center', flexWrap: 'wrap', mb: 2, rowGap: 1.25 }}>
        <TextField
          size="small"
          placeholder="Search properties, addresses"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setPage(0);
          }}
          sx={{ flex: '1 1 240px', maxWidth: 340 }}
          slotProps={{ input: { startAdornment: <InputAdornment position="start"><SearchOutlined sx={{ fontSize: 18, color: tokens.slate[400] }} /></InputAdornment> } }}
        />
        <TextField select size="small" value={typeFilter} onChange={(e) => { setTypeFilter(e.target.value as PropertyType | ''); setPage(0); }} sx={{ minWidth: 190 }}>
          <MenuItem value="">Type: All</MenuItem>
          {PROPERTY_TYPES.map((t) => (
            <MenuItem key={t} value={t}>{t}</MenuItem>
          ))}
        </TextField>
        <TextField select size="small" value={occupancyFilter} onChange={(e) => { setOccupancyFilter(e.target.value as OccupancyFilter); setPage(0); }} sx={{ minWidth: 160 }}>
          <MenuItem value="">Occupancy: All</MenuItem>
          <MenuItem value="high">90%+</MenuItem>
          <MenuItem value="mid">70–89%</MenuItem>
          <MenuItem value="low">Below 70%</MenuItem>
        </TextField>
        <TextField select size="small" value={statusFilter} onChange={(e) => { setStatusFilter(e.target.value as PropertyRowStatus | ''); setPage(0); }} sx={{ minWidth: 150 }}>
          <MenuItem value="">Status: All</MenuItem>
          <MenuItem value="Active">Active</MenuItem>
          <MenuItem value="Onboarding">Onboarding</MenuItem>
          <MenuItem value="Archived">Archived</MenuItem>
        </TextField>

        <Box sx={{ flex: 1 }} />
        <Typography sx={{ fontSize: 12.5, color: tokens.slate[500], fontFamily: tokens.fontMono, whiteSpace: 'nowrap' }}>
          {filtered.length} of {properties.length}
        </Typography>
        <ToggleButtonGroup size="small" value={density} exclusive onChange={(_e, v) => v && setDensity(v)}>
          <ToggleButton value="comfortable" aria-label="Comfortable density"><Tooltip title="Comfortable"><DensityMediumOutlined sx={{ fontSize: 17 }} /></Tooltip></ToggleButton>
          <ToggleButton value="compact" aria-label="Compact density"><Tooltip title="Compact"><DensitySmallOutlined sx={{ fontSize: 17 }} /></Tooltip></ToggleButton>
        </ToggleButtonGroup>
      </Stack>

      {selected.size > 0 && (
        <Stack direction="row" sx={{ alignItems: 'center', gap: 2, bgcolor: tokens.slate[900], color: '#fff', borderRadius: `${tokens.radiusControl}px`, px: 2, py: 1.25, mb: 2 }}>
          <Typography sx={{ fontSize: 13 }}><strong>{selected.size}</strong> selected</Typography>
          <Stack direction="row" spacing={1} sx={{ ml: 'auto' }}>
            <Button size="small" onClick={bulkExport} sx={{ color: '#fff', bgcolor: 'rgba(255,255,255,0.12)', '&:hover': { bgcolor: 'rgba(255,255,255,0.22)' } }}>Export</Button>
            <Button size="small" onClick={bulkArchive} sx={{ color: '#fff', bgcolor: 'rgba(255,255,255,0.12)', '&:hover': { bgcolor: 'rgba(255,255,255,0.22)' } }}>Archive</Button>
            <Button size="small" onClick={() => setSelected(new Set())} sx={{ color: '#fff', textDecoration: 'underline' }}>Clear</Button>
          </Stack>
        </Stack>
      )}

      <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
        <TableContainer sx={{ overflowX: 'auto' }}>
          <Table sx={{ minWidth: 860 }} size={density === 'compact' ? 'small' : 'medium'}>
            <TableHead>
              <TableRow>
                <TableCell padding="checkbox">
                  <Checkbox checked={allPageSelected} indeterminate={somePageSelected} onChange={toggleSelectAllOnPage} disabled={loading || pageRows.length === 0} />
                </TableCell>
                {(
                  [
                    ['name', 'Property'],
                    ['type', 'Type'],
                    ['units', 'Units'],
                    ['occupancyPct', 'Occupancy'],
                    ['collectedThisMonth', 'Collected this month'],
                    ['status', 'Status'],
                  ] as [SortKey, string][]
                ).map(([key, label]) => (
                  <TableCell key={key} align={key === 'units' || key === 'occupancyPct' || key === 'collectedThisMonth' ? 'right' : 'left'}>
                    <TableSortLabel active={sortKey === key} direction={sortKey === key ? sortDir : 'asc'} onClick={() => handleSort(key)}>
                      {label}
                    </TableSortLabel>
                  </TableCell>
                ))}
                <TableCell padding="checkbox" />
              </TableRow>
            </TableHead>
            <TableBody>
              {loading &&
                Array.from({ length: 5 }).map((_, i) => (
                  <TableRow key={i}>
                    <TableCell padding="checkbox"><Skeleton variant="rounded" width={18} height={18} /></TableCell>
                    <TableCell><Skeleton width="70%" /><Skeleton width="45%" /></TableCell>
                    <TableCell><Skeleton width="80%" /></TableCell>
                    <TableCell align="right"><Skeleton width={24} sx={{ ml: 'auto' }} /></TableCell>
                    <TableCell align="right"><Skeleton width={40} sx={{ ml: 'auto' }} /></TableCell>
                    <TableCell align="right"><Skeleton width={60} sx={{ ml: 'auto' }} /></TableCell>
                    <TableCell><Skeleton variant="rounded" width={64} height={22} /></TableCell>
                    <TableCell />
                  </TableRow>
                ))}

              {!loading && properties.length === 0 && (
                <TableRow>
                  <TableCell colSpan={8}>
                    <EmptyBlock
                      icon={<ApartmentOutlined sx={{ fontSize: 26 }} />}
                      title="No properties yet"
                      copy="Add your first property to start tracking units, leases, and rent collection."
                      action={<Button variant="contained" startIcon={<AddOutlined />} onClick={() => setDrawerOpen(true)}>Add Property</Button>}
                    />
                  </TableCell>
                </TableRow>
              )}

              {!loading && properties.length > 0 && filtered.length === 0 && (
                <TableRow>
                  <TableCell colSpan={8}>
                    <EmptyBlock
                      icon={<SearchOffOutlined sx={{ fontSize: 26 }} />}
                      title="No properties match your filters"
                      copy="Try a different search term, or clear your filters to see everything."
                      action={<Button variant="outlined" onClick={clearFilters} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>Clear filters</Button>}
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
                      onClick={() => setToast({ message: `Opening ${p.name}…`, severity: 'info' })}
                      sx={{
                        cursor: 'pointer',
                        ...(p.id === newRowId
                          ? { animation: 'flashNew 2.2s ease', '@keyframes flashNew': { '0%': { backgroundColor: 'rgba(15,133,119,0.14)' }, '70%': { backgroundColor: 'rgba(15,133,119,0.14)' }, '100%': { backgroundColor: 'transparent' } } }
                          : {}),
                      }}
                    >
                      <TableCell padding="checkbox" onClick={(e) => e.stopPropagation()}>
                        <Checkbox checked={selected.has(p.id)} onChange={() => toggleRow(p.id)} />
                      </TableCell>
                      <TableCell>
                        <Typography sx={{ fontWeight: 600, fontSize: 13.5 }}>{p.name}</Typography>
                        <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>{p.address}</Typography>
                      </TableCell>
                      <TableCell><Typography sx={{ fontSize: 12.5, color: tokens.slate[600] }}>{p.type}</Typography></TableCell>
                      <TableCell align="right"><Typography sx={{ fontFamily: tokens.fontMono, fontSize: 13.5 }}>{p.units}</Typography></TableCell>
                      <TableCell align="right">
                        <Stack sx={{ alignItems: 'flex-end', gap: 0.5 }}>
                          <Typography sx={{ fontFamily: tokens.fontMono, fontSize: 13.5 }} color={`${OCCUPANCY_COLOR[tier]}.main`}>{p.occupancyPct}%</Typography>
                          <LinearProgress
                            variant="determinate"
                            value={p.occupancyPct}
                            color={OCCUPANCY_COLOR[tier]}
                            sx={{ width: 64, height: 4, borderRadius: 3, bgcolor: tokens.slate[100] }}
                          />
                        </Stack>
                      </TableCell>
                      <TableCell align="right"><Typography sx={{ fontFamily: tokens.fontMono, fontSize: 13.5 }}>{currency.format(p.collectedThisMonth)}</Typography></TableCell>
                      <TableCell><Chip label={p.status} color={STATUS_COLOR[p.status]} size="small" sx={{ fontWeight: 600 }} /></TableCell>
                      <TableCell padding="checkbox" onClick={(e) => e.stopPropagation()}>
                        <IconButton size="small" onClick={(e) => setKebabRow({ id: p.id, anchor: e.currentTarget })} aria-label="Row actions">
                          <MoreVertOutlined fontSize="small" />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  );
                })}
            </TableBody>
          </Table>
        </TableContainer>

        {!loading && properties.length > 0 && filtered.length > 0 && (
          <TablePagination
            component="div"
            count={filtered.length}
            page={page}
            onPageChange={(_e, p) => setPage(p)}
            rowsPerPage={rowsPerPage}
            onRowsPerPageChange={(e) => {
              setRowsPerPage(parseInt(e.target.value, 10));
              setPage(0);
            }}
            rowsPerPageOptions={[5, 10, 25]}
          />
        )}
      </Paper>

      <Menu anchorEl={kebabRow?.anchor} open={Boolean(kebabRow)} onClose={() => setKebabRow(null)}>
        <MenuItem
          onClick={() => {
            setKebabRow(null);
            setToast({ message: 'Not available in this preview', severity: 'info' });
          }}
        >
          <VisibilityOutlined fontSize="small" style={{ marginRight: 10 }} /> View details
        </MenuItem>
        <MenuItem
          onClick={() => {
            setKebabRow(null);
            setToast({ message: 'Not available in this preview', severity: 'info' });
          }}
        >
          <EditOutlined fontSize="small" style={{ marginRight: 10 }} /> Edit property
        </MenuItem>
        <MenuItem
          onClick={() => {
            if (!kebabRow) return;
            const id = kebabRow.id;
            setKebabRow(null);
            setProperties((prev) =>
              prev.map((p) => {
                if (p.id !== id) return p;
                const nextStatus: PropertyRowStatus = p.status === 'Archived' ? 'Active' : 'Archived';
                setToast({ message: `${nextStatus === 'Archived' ? 'Archived' : 'Restored'} ${p.name}`, severity: 'success' });
                return { ...p, status: nextStatus };
              }),
            );
          }}
        >
          {properties.find((p) => p.id === kebabRow?.id)?.status === 'Archived' ? (
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

      <AddPropertyDrawer open={drawerOpen} onClose={() => setDrawerOpen(false)} onSaved={handleSaved} />

      <Dialog open={addUnitsOpen} onClose={() => setAddUnitsOpen(false)} maxWidth="xs">
        <DialogTitle component="div">
          <Typography variant="h4" sx={{ fontSize: 18 }}>Add units now?</Typography>
        </DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>
            You can set up individual units — floor plans, rent, and lease terms — right away, or come back to it later.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2.5 }}>
          <Button onClick={() => setAddUnitsOpen(false)} variant="outlined" sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>Later</Button>
          <Button
            onClick={() => {
              setAddUnitsOpen(false);
              setToast({ message: 'Opening unit setup…', severity: 'info' });
            }}
            variant="contained"
          >
            Yes, add units
          </Button>
        </DialogActions>
      </Dialog>

      <Snackbar open={Boolean(toast)} autoHideDuration={3600} onClose={() => setToast(null)} anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}>
        {toast ? (
          <Alert onClose={() => setToast(null)} severity={toast.severity} variant="filled" sx={{ borderRadius: 999 }}>
            {toast.message}
          </Alert>
        ) : undefined}
      </Snackbar>
    </Box>
  );
}

function EmptyBlock({ icon, title, copy, action }: { icon: React.ReactNode; title: string; copy: string; action: React.ReactNode }) {
  return (
    <Stack sx={{ alignItems: 'center', textAlign: 'center', py: 8, px: 3, gap: 1.5 }}>
      <Box sx={{ width: 56, height: 56, borderRadius: '50%', bgcolor: tokens.azure[50], color: tokens.azure[600], display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        {icon}
      </Box>
      <Typography variant="h5" sx={{ fontSize: 17 }}>{title}</Typography>
      <Typography sx={{ fontSize: 13.5, color: tokens.slate[500], maxWidth: 340 }}>{copy}</Typography>
      {action}
    </Stack>
  );
}
