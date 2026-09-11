import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { tokens } from '@/app/tokens';

interface MetricCardProps {
  label: string;
  value: string;
  delta: string;
  deltaTone?: 'neutral' | 'error';
}

export function MetricCard({ label, value, delta, deltaTone = 'neutral' }: MetricCardProps) {
  return (
    <Box
      sx={{
        flex: 1,
        minWidth: 200,
        p: 2.5,
        bgcolor: 'common.white',
        borderRadius: `${tokens.radiusCard}px`,
        border: `1px solid ${tokens.slate[200]}`,
        boxShadow: tokens.shadowCard,
      }}
    >
      <Typography sx={{ fontSize: 13, color: tokens.slate[500] }}>{label}</Typography>
      <Typography sx={{ fontFamily: tokens.fontMono, fontSize: 26, fontWeight: 600, color: tokens.slate[900], mt: 0.5 }}>
        {value}
      </Typography>
      <Typography
        sx={{ fontSize: 13, mt: 0.5, color: deltaTone === 'error' ? tokens.error : tokens.slate[500] }}
      >
        {delta}
      </Typography>
    </Box>
  );
}
