import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Paper from '@mui/material/Paper';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import { tokens } from '@/app/tokens';
import { useLeaseRentHistory } from '../hooks/useLeasesQueries';

const currency = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });
const dateFormatter = new Intl.DateTimeFormat('en-US', { dateStyle: 'medium' });

function formatDate(iso: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : dateFormatter.format(d);
}

interface LeaseRentHistorySectionProps {
  leaseId: string;
}

/**
 * The Lease Detail page's Rent History tab (R4) — every dated
 * amendment ever appended for this lease, most recent effective date
 * first. Read-only: a rent change only ever happens through the
 * dedicated Change Rent action on the Edit dialog, never here.
 */
export function LeaseRentHistorySection({ leaseId }: LeaseRentHistorySectionProps) {
  const { data: history, isLoading } = useLeaseRentHistory(leaseId);
  const entries = history ?? [];

  return (
    <Paper variant="outlined" sx={{ p: 3 }}>
      <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700], mb: 2 }}>Rent history</Typography>

      {isLoading ? (
        <Box sx={{ py: 4, textAlign: 'center' }}>
          <CircularProgress size={22} />
        </Box>
      ) : entries.length === 0 ? (
        <Typography sx={{ fontSize: 13.5, color: tokens.slate[400], py: 3, textAlign: 'center' }}>
          No rent changes recorded yet — this lease's rent has never been amended since it was created.
        </Typography>
      ) : (
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Effective date</TableCell>
              <TableCell align="right">Amount</TableCell>
              <TableCell>Reason</TableCell>
              <TableCell />
            </TableRow>
          </TableHead>
          <TableBody>
            {entries.map((e) => (
              <TableRow key={e.id} hover>
                <TableCell>
                  <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>{formatDate(e.effectiveDate)}</Typography>
                </TableCell>
                <TableCell align="right">
                  <Typography sx={{ fontSize: 13, color: tokens.slate[700] }}>{currency.format(e.amount)}</Typography>
                </TableCell>
                <TableCell>
                  <Typography sx={{ fontSize: 13, color: e.reason ? tokens.slate[600] : tokens.slate[400] }}>{e.reason || '—'}</Typography>
                </TableCell>
                <TableCell align="right">
                  {e.isCorrection && <Chip label="Correction" size="small" color="warning" variant="outlined" />}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </Paper>
  );
}
