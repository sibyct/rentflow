import { useNavigate } from 'react-router-dom';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { tokens } from '@/app/tokens';
import { useWorkOrderSummary } from '@/features/maintenance/hooks/useWorkOrdersQueries';
import { LoadingSpinner } from '@/shared/components';

const PRIORITY_ROWS: { key: 'emergency' | 'high' | 'medium' | 'low'; label: string; color: 'error' | 'warning' | 'info' | 'default' }[] = [
  { key: 'emergency', label: 'Emergency', color: 'error' },
  { key: 'high', label: 'High', color: 'warning' },
  { key: 'medium', label: 'Medium', color: 'info' },
  { key: 'low', label: 'Low', color: 'default' },
];

export function MaintenanceSummaryCard() {
  const navigate = useNavigate();
  const { data: summary, isLoading } = useWorkOrderSummary();

  return (
    <Paper variant="outlined" sx={{ p: 2.5, height: '100%' }}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>Maintenance</Typography>
        <Link component="button" onClick={() => navigate('/maintenance')} sx={{ fontSize: 12.5, fontWeight: 600 }}>
          View all
        </Link>
      </Stack>

      {isLoading || !summary ? (
        <LoadingSpinner label="Loading…" />
      ) : (
        <Stack spacing={2}>
          <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap', rowGap: 1 }}>
            {PRIORITY_ROWS.map((row) => (
              <Chip
                key={row.key}
                label={`${row.label}: ${summary[row.key]}`}
                color={row.color}
                size="small"
                variant={summary[row.key] > 0 ? 'filled' : 'outlined'}
                onClick={() => navigate(`/maintenance?priority=${row.key}`)}
                sx={{ fontWeight: 600, cursor: 'pointer' }}
              />
            ))}
          </Stack>
          <Box sx={{ borderTop: `1px solid ${tokens.slate[100]}`, pt: 1.5 }}>
            <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
              <Typography sx={{ fontSize: 13, color: tokens.slate[500] }}>Overdue</Typography>
              <Typography
                sx={{
                  fontSize: 13,
                  fontWeight: 700,
                  color: summary.overdue > 0 ? tokens.error : tokens.slate[700],
                }}
              >
                {summary.overdue}
              </Typography>
            </Stack>
            <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mt: 0.75 }}>
              <Typography sx={{ fontSize: 13, color: tokens.slate[500] }}>Unassigned</Typography>
              <Typography sx={{ fontSize: 13, fontWeight: 700, color: tokens.slate[700] }}>{summary.unassigned}</Typography>
            </Stack>
          </Box>
        </Stack>
      )}
    </Paper>
  );
}
