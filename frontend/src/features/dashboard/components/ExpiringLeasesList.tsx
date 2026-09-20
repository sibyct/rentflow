import { Link as RouterLink, useNavigate } from 'react-router-dom';
import Box from '@mui/material/Box';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { tokens } from '@/app/tokens';
import { useLeasesPortfolio } from '@/features/leases/hooks/useLeasesQueries';
import { EmptyState, LoadingSpinner } from '@/shared/components';
import DescriptionOutlined from '@mui/icons-material/DescriptionOutlined';

const dateFormatter = new Intl.DateTimeFormat('en-US', { dateStyle: 'medium' });

function formatDate(iso: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : dateFormatter.format(d);
}

export function ExpiringLeasesList() {
  const navigate = useNavigate();
  const { data, isLoading } = useLeasesPortfolio({ status: 'expiring_soon', sort: 'endDate', order: 'asc', limit: 5, offset: 0 });
  const leases = data?.leases ?? [];
  const total = data?.total ?? 0;

  return (
    <Paper variant="outlined" sx={{ p: 2.5 }}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>Leases Expiring Soon</Typography>
        <Link component={RouterLink} to="/leases?status=expiring_soon" sx={{ fontSize: 12.5, fontWeight: 600 }}>
          View all
        </Link>
      </Stack>

      {isLoading ? (
        <LoadingSpinner label="Loading…" />
      ) : leases.length === 0 ? (
        <EmptyState icon={DescriptionOutlined} title="No leases expiring soon" description="Nothing in the next 30 days." />
      ) : (
        <Stack spacing={0}>
          {leases.map((l) => (
            <Stack
              key={l.id}
              direction="row"
              onClick={() => navigate(`/leases/${l.id}`)}
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
                <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[900] }}>{l.primaryResidentName}</Typography>
                <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>
                  {l.propertyName} / {l.unitName}
                </Typography>
              </Box>
              <Typography sx={{ fontSize: 12.5, color: tokens.slate[600], whiteSpace: 'nowrap' }}>{formatDate(l.endDate)}</Typography>
            </Stack>
          ))}
          {total > leases.length && (
            <Typography sx={{ fontSize: 11.5, color: tokens.slate[400], mt: 1 }}>+{total - leases.length} more</Typography>
          )}
        </Stack>
      )}
    </Paper>
  );
}
