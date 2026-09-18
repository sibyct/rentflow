import { useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import Switch from '@mui/material/Switch';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import AddOutlined from '@mui/icons-material/AddOutlined';
import AutorenewOutlined from '@mui/icons-material/AutorenewOutlined';
import DeleteOutlineOutlined from '@mui/icons-material/DeleteOutlineOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';
import EventRepeatOutlined from '@mui/icons-material/EventRepeatOutlined';
import { tokens } from '@/app/tokens';
import { EmptyState } from '@/shared/components';
import { useProperties } from '@/features/properties/hooks/usePropertiesQueries';
import {
  useDeleteRecurringRule,
  useGenerateWorkOrderNow,
  useRecurringRules,
  useSetRecurringRuleActive,
} from '../hooks/useRecurringRulesQueries';
import { WORK_ORDER_CATEGORY_LABELS, type RecurringRuleRow } from '../types';
import { RecurringRuleDialog } from './RecurringRuleDialog';

const dateFormatter = new Intl.DateTimeFormat('en-US', { dateStyle: 'medium' });

function formatDate(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : dateFormatter.format(d);
}

function isDue(iso: string): boolean {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return false;
  return d <= new Date();
}

interface RecurringRulesTabProps {
  onWorkOrderGenerated: (workOrderId: string) => void;
}

/** The Scheduled tab: recurring maintenance rules and the work orders they generate. Generation is manager-triggered ("Generate now") rather than automatic — there's no background scheduler in this app yet; see the backend's RecurringRule doc comment. */
export function RecurringRulesTab({ onWorkOrderGenerated }: RecurringRulesTabProps) {
  const { data: rules, isLoading } = useRecurringRules();
  const { data: propertiesData } = useProperties({ limit: 100, offset: 0 });
  const properties = propertiesData?.properties ?? [];

  const [dialogRule, setDialogRule] = useState<RecurringRuleRow | 'new' | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<RecurringRuleRow | null>(null);
  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);

  const setActive = useSetRecurringRuleActive();
  const deleteRule = useDeleteRecurringRule();
  const generateNow = useGenerateWorkOrderNow();

  const rows = rules ?? [];

  return (
    <Box>
      <Stack direction="row" sx={{ justifyContent: 'flex-end', mb: 2 }}>
        <Button size="small" variant="contained" startIcon={<AddOutlined />} onClick={() => setDialogRule('new')}>
          New Recurring Rule
        </Button>
      </Stack>

      <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
        {!isLoading && rows.length === 0 ? (
          <EmptyState
            icon={EventRepeatOutlined}
            title="No recurring maintenance scheduled"
            description="Set up a rule (e.g. HVAC filter change every 3 months) to auto-generate work orders on a schedule."
            action={
              <Button variant="contained" startIcon={<AddOutlined />} onClick={() => setDialogRule('new')}>
                New Recurring Rule
              </Button>
            }
          />
        ) : (
          <TableContainer sx={{ overflowX: 'auto' }}>
            <Table size="small" sx={{ minWidth: 760 }}>
              <TableHead>
                <TableRow>
                  <TableCell>Title</TableCell>
                  <TableCell>Property / Unit</TableCell>
                  <TableCell>Category</TableCell>
                  <TableCell>Frequency</TableCell>
                  <TableCell>Next due</TableCell>
                  <TableCell>Active</TableCell>
                  <TableCell padding="checkbox" />
                </TableRow>
              </TableHead>
              <TableBody>
                {rows.map((r) => (
                  <TableRow key={r.id} hover>
                    <TableCell>
                      <Typography sx={{ fontWeight: 600, fontSize: 13.5 }}>{r.title}</Typography>
                    </TableCell>
                    <TableCell>
                      <Link component={RouterLink} to={`/properties/${r.propertyId}`} sx={{ fontSize: 13, fontWeight: 600, display: 'block' }}>
                        {r.propertyName}
                      </Link>
                      {r.unitName && <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>{r.unitName}</Typography>}
                    </TableCell>
                    <TableCell>
                      <Typography sx={{ fontSize: 12.5, color: tokens.slate[600] }}>{WORK_ORDER_CATEGORY_LABELS[r.category]}</Typography>
                    </TableCell>
                    <TableCell>
                      <Typography sx={{ fontSize: 12.5, color: tokens.slate[600] }}>Every {r.frequencyInterval} {r.frequencyUnit}</Typography>
                    </TableCell>
                    <TableCell>
                      <Typography sx={{ fontSize: 12.5, fontWeight: isDue(r.nextDueDate) ? 700 : 400, color: isDue(r.nextDueDate) ? tokens.warningInk : tokens.slate[600] }}>
                        {formatDate(r.nextDueDate)}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Switch
                        size="small"
                        checked={r.active}
                        onChange={(e) => setActive.mutate({ id: r.id, active: e.target.checked })}
                      />
                    </TableCell>
                    <TableCell padding="checkbox">
                      <Stack direction="row" spacing={0.25} sx={{ justifyContent: 'flex-end' }}>
                        <Tooltip title="Generate work order now">
                          <span>
                            <IconButton
                              size="small"
                              disabled={generateNow.isPending}
                              onClick={() =>
                                generateNow.mutate(r.id, {
                                  onSuccess: (res) => {
                                    setToast({ message: 'Work order generated', severity: 'success' });
                                    onWorkOrderGenerated(res.workOrderId);
                                  },
                                  onError: () => setToast({ message: 'Something went wrong. Please try again.', severity: 'error' }),
                                })
                              }
                              aria-label={`Generate work order from ${r.title}`}
                            >
                              <AutorenewOutlined fontSize="small" />
                            </IconButton>
                          </span>
                        </Tooltip>
                        <Tooltip title="Edit">
                          <IconButton size="small" onClick={() => setDialogRule(r)} aria-label={`Edit ${r.title}`}>
                            <EditOutlined fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="Delete">
                          <IconButton size="small" onClick={() => setDeleteTarget(r)} aria-label={`Delete ${r.title}`}>
                            <DeleteOutlineOutlined fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </Stack>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Paper>

      <RecurringRuleDialog
        open={dialogRule !== null}
        onClose={() => setDialogRule(null)}
        properties={properties}
        rule={dialogRule === 'new' ? undefined : (dialogRule ?? undefined)}
        onSaved={() => {
          setDialogRule(null);
          setToast({ message: dialogRule === 'new' ? 'Recurring rule created' : 'Recurring rule updated', severity: 'success' });
        }}
      />

      <Dialog open={Boolean(deleteTarget)} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete this recurring rule?</DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>
            {deleteTarget ? `"${deleteTarget.title}"` : 'This rule'} will be permanently removed — existing work orders it already generated are not affected.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 0 }}>
          <Button variant="text" onClick={() => setDeleteTarget(null)} sx={{ color: tokens.slate[600] }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color="error"
            disabled={deleteRule.isPending}
            onClick={() => {
              if (!deleteTarget) return;
              deleteRule.mutate(deleteTarget.id, { onSuccess: () => setDeleteTarget(null) });
            }}
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      <Snackbar open={Boolean(toast)} autoHideDuration={3200} onClose={() => setToast(null)} anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}>
        {toast ? (
          <Alert onClose={() => setToast(null)} severity={toast.severity} variant="filled" sx={{ borderRadius: 999 }}>
            {toast.message}
          </Alert>
        ) : undefined}
      </Snackbar>
    </Box>
  );
}
