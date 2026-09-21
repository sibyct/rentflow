import { Link as RouterLink, useNavigate } from 'react-router-dom';
import Box from '@mui/material/Box';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import ApartmentOutlined from '@mui/icons-material/ApartmentOutlined';
import { tokens } from '@/app/tokens';
import { useUnitsPortfolio } from '@/features/units/hooks/useUnitsQueries';
import { EmptyState, LoadingSpinner } from '@/shared/components';

function daysVacant(iso: string): string {
  if (!iso) return '—';
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return '—';
  const days = Math.max(0, Math.round((Date.now() - then) / (1000 * 60 * 60 * 24)));
  return `${days} day${days === 1 ? '' : 's'}`;
}

export function VacantUnitsList() {
  const navigate = useNavigate();
  // vacatedAt isn't a server-side sort key (see UnitPortfolioSortKey) —
  // sorted client-side after fetching, same as the plan's scoped-down
  // first pass.
  const { data, isLoading } = useUnitsPortfolio({ status: 'vacant', sort: 'propertyName', order: 'asc', limit: 25, offset: 0 });
  const units = [...(data?.units ?? [])].sort((a, b) => {
    if (!a.vacatedAt) return 1;
    if (!b.vacatedAt) return -1;
    return a.vacatedAt.localeCompare(b.vacatedAt);
  });
  const total = data?.total ?? 0;
  const visible = units.slice(0, 5);

  return (
    <Paper variant="outlined" sx={{ p: 2.5 }}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>Vacant Units</Typography>
        <Link component={RouterLink} to="/units?status=vacant" sx={{ fontSize: 12.5, fontWeight: 600 }}>
          View all
        </Link>
      </Stack>

      {isLoading ? (
        <LoadingSpinner label="Loading…" />
      ) : visible.length === 0 ? (
        <EmptyState icon={ApartmentOutlined} title="No vacant units" description="Every unit in your portfolio is occupied." />
      ) : (
        <Stack spacing={0}>
          {visible.map((u) => (
            <Stack
              key={u.id}
              direction="row"
              onClick={() => navigate(`/units?status=vacant&property=${u.propertyId}`)}
              sx={{
                alignItems: 'center',
                justifyContent: 'space-between',
                py: 1,
                borderBottom: `1px solid ${tokens.slate[100]}`,
                cursor: 'pointer',
                '&:hover': { bgcolor: tokens.slate[50] },
                '&:last-of-type': { borderBottom: 'none' },
              }}
            >
              <Box>
                <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[900] }}>{u.unitName}</Typography>
                <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>{u.propertyName}</Typography>
              </Box>
              <Typography sx={{ fontSize: 12.5, color: tokens.slate[600], whiteSpace: 'nowrap' }}>{daysVacant(u.vacatedAt)}</Typography>
            </Stack>
          ))}
          {total > visible.length && (
            <Typography sx={{ fontSize: 11.5, color: tokens.slate[400], mt: 1 }}>+{total - visible.length} more</Typography>
          )}
        </Stack>
      )}
    </Paper>
  );
}
