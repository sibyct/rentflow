import { useMemo, useState } from 'react';
import Box from '@mui/material/Box';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import HistoryOutlined from '@mui/icons-material/HistoryOutlined';
import { tokens } from '@/app/tokens';
import { EmptyState, ErrorMessage, LoadingSpinner } from '@/shared/components';
import { useStaffAuditLog } from '../hooks/useStaffQueries';
import { AUDIT_EVENT_LABELS, type AuditEventKind } from '../types';

const dateTimeFormatter = new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' });
const AUDIT_LOG_LIMIT = 100;

const EVENT_FILTERS: (AuditEventKind | '')[] = ['', 'invite_sent', 'invite_resent', 'invite_accepted', 'role_or_access_changed', 'deactivated', 'reactivated'];

export function AuditLogTab() {
  const { data, isLoading, isError, error, refetch } = useStaffAuditLog(AUDIT_LOG_LIMIT, 0);
  const [eventFilter, setEventFilter] = useState<AuditEventKind | ''>('');

  const rows = useMemo(() => {
    const entries = data?.entries ?? [];
    return eventFilter ? entries.filter((e) => e.event === eventFilter) : entries;
  }, [data, eventFilter]);

  return (
    <Box>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
        <TextField select size="small" value={eventFilter} onChange={(e) => setEventFilter(e.target.value as AuditEventKind | '')} sx={{ minWidth: 200 }}>
          <MenuItem value="">Event: All</MenuItem>
          {EVENT_FILTERS.filter(Boolean).map((event) => (
            <MenuItem key={event} value={event}>
              {AUDIT_EVENT_LABELS[event as AuditEventKind]}
            </MenuItem>
          ))}
        </TextField>
        <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>Kept for 7 years</Typography>
      </Stack>

      {isError ? (
        <ErrorMessage error={error} onRetry={() => void refetch()} />
      ) : isLoading ? (
        <Box sx={{ py: 8 }}>
          <LoadingSpinner label="Loading activity…" />
        </Box>
      ) : (
        <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
          <TableContainer sx={{ overflowX: 'auto' }}>
            <Table size="small" sx={{ minWidth: 720 }}>
              {rows.length > 0 && (
                <TableHead>
                  <TableRow>
                    <TableCell>When</TableCell>
                    <TableCell>Event</TableCell>
                    <TableCell>Change</TableCell>
                    <TableCell>Changed by</TableCell>
                  </TableRow>
                </TableHead>
              )}
              <TableBody>
                {rows.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4}>
                      <EmptyState icon={HistoryOutlined} title="No matching activity" description="Nothing has happened for this event type yet." />
                    </TableCell>
                  </TableRow>
                ) : (
                  rows.map((entry) => {
                    const fields = Object.entries(entry.changes);
                    return (
                      <TableRow key={entry.id} hover>
                        <TableCell>
                          <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>{dateTimeFormatter.format(new Date(entry.when))}</Typography>
                        </TableCell>
                        <TableCell>
                          <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{AUDIT_EVENT_LABELS[entry.event]}</Typography>
                          <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>{entry.subjectName}</Typography>
                        </TableCell>
                        <TableCell>
                          {fields.length > 0 ? (
                            <Stack spacing={0.5}>
                              {fields.map(([field, change]) => (
                                <Stack key={field} direction="row" spacing={0.75} sx={{ alignItems: 'center' }}>
                                  {fields.length > 1 && (
                                    <Typography sx={{ fontSize: 11, color: tokens.slate[400], minWidth: 88 }}>
                                      {field === 'role' ? 'Role' : field === 'property_access' ? 'Property access' : field}
                                    </Typography>
                                  )}
                                  <Box sx={{ fontSize: 11.5, color: tokens.slate[600], bgcolor: tokens.slate[100], borderRadius: '4px', px: 0.8, py: 0.2 }}>
                                    {change.old}
                                  </Box>
                                  <Typography sx={{ fontSize: 12, color: tokens.slate[400] }}>&rarr;</Typography>
                                  <Box sx={{ fontSize: 11.5, fontWeight: 600, color: tokens.azure[600], bgcolor: tokens.azure[50], borderRadius: '4px', px: 0.8, py: 0.2 }}>
                                    {change.new}
                                  </Box>
                                </Stack>
                              ))}
                            </Stack>
                          ) : (
                            <Typography sx={{ fontSize: 13, color: tokens.slate[400] }}>&mdash;</Typography>
                          )}
                        </TableCell>
                        <TableCell>
                          <Typography sx={{ fontSize: 13 }}>{entry.changedBy}</Typography>
                        </TableCell>
                      </TableRow>
                    );
                  })
                )}
              </TableBody>
            </Table>
          </TableContainer>
        </Paper>
      )}
    </Box>
  );
}
