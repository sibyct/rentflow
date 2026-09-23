import { useState } from 'react';
import { useNavigate, Link as RouterLink } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Rating from '@mui/material/Rating';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Tabs from '@mui/material/Tabs';
import Typography from '@mui/material/Typography';
import DeleteOutlineOutlined from '@mui/icons-material/DeleteOutlineOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import { useProperties } from '@/features/properties/hooks/usePropertiesQueries';
import { useWorkOrders } from '@/features/maintenance/hooks/useWorkOrdersQueries';
import { WORK_ORDER_PRIORITY_LABELS, WORK_ORDER_STATUS_LABELS } from '@/features/maintenance/types';
import { ERROR_STATE_PRESETS, ErrorState, LoadingSpinner } from '@/shared/components';
import { attachmentsApi } from '@/shared/lib/attachmentsApi';
import { useDeleteVendor, useVendor, useVendorSpendSummary } from '../hooks/useVendorsQueries';
import { INSURANCE_STATUS_LABELS, VENDOR_PAYMENT_TERMS_LABELS, VENDOR_RATE_TYPE_LABELS, WORK_ORDER_CATEGORY_LABELS, type InsuranceStatus, type VendorDetail } from '../types';
import { VendorFormDialog } from './VendorFormDialog';

const INSURANCE_COLOR: Record<InsuranceStatus, 'success' | 'warning' | 'error' | 'default'> = {
  valid: 'success',
  expiring_soon: 'warning',
  expired: 'error',
  unknown: 'default',
};

const currency = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });
const dateFormatter = new Intl.DateTimeFormat('en-US', { dateStyle: 'medium' });

function formatDate(iso: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : dateFormatter.format(d);
}

interface VendorDetailScreenProps {
  vendorId: string;
}

export function VendorDetailScreen({ vendorId }: VendorDetailScreenProps) {
  const navigate = useNavigate();
  const { data: vendor, isLoading, isError, error, refetch } = useVendor(vendorId);
  const deleteVendor = useDeleteVendor();
  const { data: propertiesData } = useProperties({ limit: 100, offset: 0 });
  const properties = propertiesData?.properties ?? [];

  const [tab, setTab] = useState<'overview' | 'work-orders' | 'financials' | 'notes'>('overview');
  const [editOpen, setEditOpen] = useState(false);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);

  if (isLoading) {
    return (
      <Box sx={{ py: 8 }}>
        <LoadingSpinner label="Loading vendor…" />
      </Box>
    );
  }

  if (isError) {
    if (error instanceof ApiError && error.status === 404) {
      return (
        <ErrorState
          {...ERROR_STATE_PRESETS.notFound}
          title="We can't find that vendor"
          message="It may have been removed, or the link is wrong. Check the address bar, or head back to Vendors."
          primaryAction={{ label: 'Back to Vendors', href: '/vendors' }}
        />
      );
    }
    return <ErrorState {...ERROR_STATE_PRESETS.serverError} primaryAction={{ label: 'Try again', onClick: () => void refetch() }} />;
  }

  if (!vendor) return null;

  function handleDelete() {
    if (!vendor) return;
    deleteVendor.mutate(vendor.id, {
      onSuccess: () => navigate('/vendors'),
      onError: () => setToast({ message: 'Something went wrong. Please try again.', severity: 'error' }),
    });
  }

  return (
    <Box>
      <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', fontSize: 13, mb: 2, flexWrap: 'wrap' }}>
        <Link component={RouterLink} to="/dashboard" sx={{ color: tokens.slate[500], textDecoration: 'none', '&:hover': { color: tokens.slate[700] } }}>
          Portfolio
        </Link>
        <Typography sx={{ fontSize: 13, color: tokens.slate[400] }}>/</Typography>
        <Link component={RouterLink} to="/vendors" sx={{ color: tokens.slate[500], textDecoration: 'none', '&:hover': { color: tokens.slate[700] } }}>
          Vendors
        </Link>
        <Typography sx={{ fontSize: 13, color: tokens.slate[400] }}>/</Typography>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>{vendor.companyName}</Typography>
      </Stack>

      <Stack direction="row" sx={{ alignItems: 'flex-start', justifyContent: 'space-between', gap: 2, flexWrap: 'wrap', mb: 1 }}>
        <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
          <Typography variant="h3">{vendor.companyName}</Typography>
          <Chip label={vendor.active ? 'Active' : 'Inactive'} color={vendor.active ? 'success' : 'default'} size="small" sx={{ fontWeight: 600 }} />
          <Chip label={INSURANCE_STATUS_LABELS[vendor.insuranceStatus]} color={INSURANCE_COLOR[vendor.insuranceStatus]} size="small" sx={{ fontWeight: 600 }} />
        </Stack>
        <Stack direction="row" spacing={1.5}>
          <Button variant="outlined" startIcon={<EditOutlined />} onClick={() => setEditOpen(true)} sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}>
            Edit
          </Button>
          <Button variant="outlined" color="error" startIcon={<DeleteOutlineOutlined />} onClick={() => setConfirmDeleteOpen(true)} sx={{ borderColor: tokens.slate[300] }}>
            Delete
          </Button>
        </Stack>
      </Stack>

      <Stack direction="row" spacing={0.5} sx={{ flexWrap: 'wrap', rowGap: 0.5, mb: 2.5 }}>
        {vendor.categories.map((c) => (
          <Chip key={c} label={WORK_ORDER_CATEGORY_LABELS[c]} size="small" variant="outlined" />
        ))}
      </Stack>

      <Stack direction="row" spacing={2} sx={{ flexWrap: 'wrap', mb: 2.5 }}>
        <StatTile label="Open jobs" value={String(vendor.openWorkOrders)} />
        <StatTile
          label="Average rating"
          value={vendor.averageRating != null ? vendor.averageRating.toFixed(1) : '—'}
          secondary={vendor.averageRating != null ? undefined : 'Not yet rated'}
        />
        <StatTile
          label="Properties served"
          value={vendor.servesAllProperties ? 'All' : String(vendor.propertiesServedCount)}
        />
      </Stack>

      <Tabs value={tab} onChange={(_e, v) => setTab(v)} sx={{ mb: 2.5, borderBottom: `1px solid ${tokens.slate[100]}` }}>
        <Tab label="Overview" value="overview" />
        <Tab label="Work Orders" value="work-orders" />
        <Tab label="Financials" value="financials" />
        <Tab label="Notes" value="notes" />
      </Tabs>

      {tab === 'overview' && <OverviewTab vendor={vendor} />}
      {tab === 'work-orders' && <WorkOrdersTab vendorId={vendor.id} vendorName={vendor.companyName} />}
      {tab === 'financials' && <FinancialsTab vendorId={vendor.id} />}
      {tab === 'notes' && <NotesTab notes={vendor.internalNotes} />}

      <VendorFormDialog
        open={editOpen}
        onClose={() => setEditOpen(false)}
        properties={properties}
        vendorId={vendorId}
        onSaved={() => {
          setEditOpen(false);
          setToast({ message: 'Vendor updated successfully', severity: 'success' });
        }}
      />

      <Dialog open={confirmDeleteOpen} onClose={() => setConfirmDeleteOpen(false)}>
        <DialogTitle>Delete this vendor?</DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>
            &quot;{vendor.companyName}&quot; will be permanently removed. Work orders already assigned to it keep their history but lose the vendor link. This can&apos;t be undone.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 0 }}>
          <Button variant="text" onClick={() => setConfirmDeleteOpen(false)} sx={{ color: tokens.slate[600] }}>
            Cancel
          </Button>
          <Button variant="contained" color="error" onClick={handleDelete} disabled={deleteVendor.isPending}>
            Delete vendor
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

function OverviewTab({ vendor }: { vendor: VendorDetail }) {
  return (
    <Stack direction={{ xs: 'column', md: 'row' }} spacing={2.5} sx={{ alignItems: 'flex-start' }}>
      <Stack spacing={2.5} sx={{ flex: '2 1 480px', minWidth: 0, width: '100%' }}>
        <Paper variant="outlined" sx={{ p: 3 }}>
          <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 2 }}>Contact</Typography>
          <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)' }, gap: 2 }}>
            <Field label="Contact person" value={vendor.contactPerson} />
            <Field label="Phone" value={vendor.phone} />
            <Field label="Email" value={vendor.email} />
            <Field label="Address" value={vendor.address} />
          </Box>

          <Box sx={{ borderTop: `1px solid ${tokens.slate[100]}`, mt: 3, pt: 2.5 }}>
            <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 1.5 }}>Compliance</Typography>
            <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)' }, gap: 2 }}>
              <Field label="Insurance expiry" value={formatDate(vendor.insuranceExpiry)} />
              <Field label="License number" value={vendor.licenseNumber} />
              <Field label="License expiry" value={formatDate(vendor.licenseExpiry)} />
              <AttachmentField label="Certificate of insurance" attachmentId={vendor.coiAttachmentId} />
              <AttachmentField label="Tax document" attachmentId={vendor.taxDocAttachmentId} />
            </Box>
          </Box>

          <Box sx={{ borderTop: `1px solid ${tokens.slate[100]}`, mt: 3, pt: 2.5 }}>
            <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 1.5 }}>Billing</Typography>
            <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(3, 1fr)' }, gap: 2 }}>
              <Field label="Rate type" value={vendor.rateType ? VENDOR_RATE_TYPE_LABELS[vendor.rateType] : ''} />
              <Field label="Rate amount" value={vendor.rateAmount != null ? currency.format(vendor.rateAmount) : ''} />
              <Field label="Payment terms" value={vendor.paymentTerms ? VENDOR_PAYMENT_TERMS_LABELS[vendor.paymentTerms] : ''} />
            </Box>
          </Box>
        </Paper>
      </Stack>

      <Stack spacing={2.5} sx={{ flex: '1 1 260px', minWidth: 0, width: '100%' }}>
        <Paper variant="outlined" sx={{ p: 3 }}>
          <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 2 }}>Service scope</Typography>
          {vendor.servesAllProperties ? (
            <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>Serves every property in the portfolio.</Typography>
          ) : (
            <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>
              Scoped to {vendor.propertiesServedCount} propert{vendor.propertiesServedCount === 1 ? 'y' : 'ies'}.
            </Typography>
          )}
        </Paper>

        <Paper variant="outlined" sx={{ p: 3 }}>
          <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 2 }}>Performance</Typography>
          <Stack spacing={1.5}>
            <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
              <Typography sx={{ fontSize: 13, color: tokens.slate[500] }}>Average rating</Typography>
              {vendor.averageRating != null ? (
                <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
                  <Rating value={vendor.averageRating} precision={0.5} readOnly size="small" />
                </Stack>
              ) : (
                <Typography sx={{ fontSize: 13, color: tokens.slate[400] }}>Not yet rated</Typography>
              )}
            </Stack>
            <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
              <Typography sx={{ fontSize: 13, color: tokens.slate[500] }}>Open jobs</Typography>
              <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>{vendor.openWorkOrders}</Typography>
            </Stack>
          </Stack>
        </Paper>
      </Stack>
    </Stack>
  );
}

function WorkOrdersTab({ vendorId, vendorName }: { vendorId: string; vendorName: string }) {
  const { data, isLoading } = useWorkOrders({ vendorId, limit: 50, offset: 0 });
  const workOrders = data?.workOrders ?? [];

  if (isLoading) {
    return (
      <Box sx={{ py: 4 }}>
        <LoadingSpinner label="Loading work orders…" />
      </Box>
    );
  }

  return (
    <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
      {workOrders.length === 0 ? (
        <Box sx={{ p: 4, textAlign: 'center' }}>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[500] }}>No work orders assigned to this vendor yet.</Typography>
        </Box>
      ) : (
        <TableContainer sx={{ overflowX: 'auto' }}>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Title</TableCell>
                <TableCell>Property / Unit</TableCell>
                <TableCell>Priority</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Due</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {workOrders.map((w) => (
                <TableRow key={w.id} hover>
                  <TableCell>
                    <Link component={RouterLink} to={`/maintenance?vendor_id=${vendorId}&vendor_name=${encodeURIComponent(vendorName)}`} sx={{ fontSize: 13, fontWeight: 600 }}>
                      {w.title}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontSize: 12.5, color: tokens.slate[600] }}>
                      {w.propertyName}
                      {w.unitName ? ` / ${w.unitName}` : ''}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontSize: 12.5, color: tokens.slate[600] }}>{WORK_ORDER_PRIORITY_LABELS[w.priority]}</Typography>
                  </TableCell>
                  <TableCell>
                    <Chip label={WORK_ORDER_STATUS_LABELS[w.status]} size="small" variant="outlined" sx={{ fontWeight: 600 }} />
                  </TableCell>
                  <TableCell>
                    <Typography sx={{ fontSize: 12.5, color: tokens.slate[600] }}>{formatDate(w.dueDate)}</Typography>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Paper>
  );
}

function FinancialsTab({ vendorId }: { vendorId: string }) {
  const { data, isLoading } = useVendorSpendSummary(vendorId);

  if (isLoading || !data) {
    return (
      <Box sx={{ py: 4 }}>
        <LoadingSpinner label="Loading spend summary…" />
      </Box>
    );
  }

  return (
    <Paper variant="outlined" sx={{ p: 3 }}>
      <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 2 }}>Spend</Typography>
      <Typography sx={{ fontSize: 12, color: tokens.slate[500], mb: 2.5 }}>
        Computed from the actual cost of work orders assigned to this vendor — there is no separate invoicing ledger.
      </Typography>
      <Stack direction="row" spacing={2} sx={{ flexWrap: 'wrap' }}>
        <StatTile label="This month" value={currency.format(data.thisMonth)} />
        <StatTile label="Year to date" value={currency.format(data.yearToDate)} />
      </Stack>
    </Paper>
  );
}

function NotesTab({ notes }: { notes: string }) {
  return (
    <Paper variant="outlined" sx={{ p: 3 }}>
      <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 1 }}>Internal notes</Typography>
      <Typography sx={{ fontSize: 13.5, color: notes ? tokens.slate[700] : tokens.slate[400], whiteSpace: 'pre-wrap' }}>
        {notes || 'No notes yet.'}
      </Typography>
    </Paper>
  );
}

function StatTile({ label, value, secondary }: { label: string; value: string; secondary?: string }) {
  return (
    <Paper variant="outlined" sx={{ p: 2, flex: '1 1 180px', minWidth: 160 }}>
      <Typography sx={{ fontSize: 11, fontWeight: 600, letterSpacing: '0.04em', color: tokens.slate[500], textTransform: 'uppercase', mb: 0.5 }}>
        {label}
      </Typography>
      <Typography sx={{ fontSize: 18, fontWeight: 600, color: tokens.slate[900] }}>{value}</Typography>
      {secondary && <Typography sx={{ fontSize: 12, color: tokens.slate[400], mt: 0.25 }}>{secondary}</Typography>}
    </Paper>
  );
}

function AttachmentField({ label, attachmentId }: { label: string; attachmentId: string }) {
  const [error, setError] = useState<string | null>(null);

  async function view() {
    try {
      const { url } = await attachmentsApi.getUrl(attachmentId);
      window.open(url, '_blank', 'noopener,noreferrer');
    } catch {
      setError('Could not open the file.');
    }
  }

  return (
    <Box>
      <Typography sx={{ fontSize: 11, fontWeight: 600, letterSpacing: '0.04em', color: tokens.slate[500], textTransform: 'uppercase' }}>
        {label}
      </Typography>
      {attachmentId ? (
        <Link component="button" type="button" onClick={() => void view()} sx={{ fontSize: 13.5, mt: 0.25, display: 'block' }}>
          View file
        </Link>
      ) : (
        <Typography sx={{ fontSize: 13.5, color: tokens.slate[400], mt: 0.25 }}>—</Typography>
      )}
      {error && <Typography sx={{ fontSize: 11.5, color: tokens.error, mt: 0.25 }}>{error}</Typography>}
    </Box>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <Box>
      <Typography sx={{ fontSize: 11, fontWeight: 600, letterSpacing: '0.04em', color: tokens.slate[500], textTransform: 'uppercase' }}>
        {label}
      </Typography>
      <Typography sx={{ fontSize: 13.5, color: value ? tokens.slate[700] : tokens.slate[400], mt: 0.25 }}>{value || '—'}</Typography>
    </Box>
  );
}
