import { useNavigate } from 'react-router-dom';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import VerifiedUserOutlined from '@mui/icons-material/VerifiedUserOutlined';
import { tokens } from '@/app/tokens';
import { useVendors } from '@/features/vendors/hooks/useVendorsQueries';
import { EmptyState, LoadingSpinner } from '@/shared/components';

function daysUntil(iso: string): number | null {
  if (!iso) return null;
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return null;
  return Math.round((then - Date.now()) / (1000 * 60 * 60 * 24));
}

export function ComplianceAlertsCard() {
  const navigate = useNavigate();
  const { data, isLoading } = useVendors({ insuranceStatus: 'expiring_soon', limit: 10, offset: 0 });
  const vendors = data?.vendors ?? [];

  return (
    <Paper variant="outlined" sx={{ p: 2.5, height: '100%' }}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>Compliance Alerts</Typography>
        <Link component="button" onClick={() => navigate('/vendors')} sx={{ fontSize: 12.5, fontWeight: 600 }}>
          View all
        </Link>
      </Stack>

      {isLoading ? (
        <LoadingSpinner label="Loading…" />
      ) : vendors.length === 0 ? (
        <EmptyState icon={VerifiedUserOutlined} title="Nothing expiring soon" description="No vendor insurance is due to expire in the next 30 days." />
      ) : (
        <Stack spacing={0}>
          {vendors.map((v) => {
            const days = daysUntil(v.insuranceExpiry);
            return (
              <Stack
                key={v.id}
                direction="row"
                onClick={() => navigate(`/vendors/${v.id}`)}
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
                  <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[900] }}>{v.companyName}</Typography>
                  <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>Insurance</Typography>
                </Box>
                <Chip
                  label={days !== null ? `${days} day${days === 1 ? '' : 's'}` : '—'}
                  size="small"
                  color="warning"
                  sx={{ fontWeight: 600 }}
                />
              </Stack>
            );
          })}
        </Stack>
      )}
    </Paper>
  );
}
