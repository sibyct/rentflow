import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import Skeleton from '@mui/material/Skeleton';
import Typography from '@mui/material/Typography';
import { tokens } from '@/app/tokens';

interface MetricCardProps {
  label: string;
  value: string;
  delta: string;
  deltaTone?: 'neutral' | 'error' | 'success';
  /** Present = the whole card is clickable (e.g. Dashboard KPI cards routing into their module). Absent = static, as every pre-Dashboard usage already is. */
  onClick?: () => void;
  /** Shows a skeleton in place of value/delta — for a card whose data hasn't loaded yet, so the Dashboard's sections can render progressively rather than blocking on the slowest query. */
  loading?: boolean;
}

const cardSx = {
  flex: 1,
  minWidth: 200,
  p: 2.5,
  bgcolor: 'common.white',
  borderRadius: `${tokens.radiusCard}px`,
  border: `1px solid ${tokens.slate[200]}`,
  boxShadow: tokens.shadowCard,
} as const;

export function MetricCard({ label, value, delta, deltaTone = 'neutral', onClick, loading = false }: MetricCardProps) {
  const content = (
    <>
      <Typography sx={{ fontSize: 13, color: tokens.slate[500] }}>{label}</Typography>
      {loading ? (
        <>
          <Skeleton variant="text" width={100} height={34} sx={{ mt: 0.5 }} />
          <Skeleton variant="text" width={140} height={20} sx={{ mt: 0.5 }} />
        </>
      ) : (
        <>
          <Typography sx={{ fontFamily: tokens.fontMono, fontSize: 26, fontWeight: 600, color: tokens.slate[900], mt: 0.5 }}>
            {value}
          </Typography>
          <Typography
            sx={{
              fontSize: 13,
              mt: 0.5,
              color:
                deltaTone === 'error' ? tokens.error : deltaTone === 'success' ? tokens.brand.green : tokens.slate[500],
            }}
          >
            {delta}
          </Typography>
        </>
      )}
    </>
  );

  if (onClick) {
    return (
      <ButtonBase
        onClick={onClick}
        focusRipple
        sx={{
          ...cardSx,
          display: 'block',
          textAlign: 'left',
          transition: 'box-shadow 0.15s, transform 0.15s',
          '&:hover': { boxShadow: tokens.shadowPopover, transform: 'translateY(-1px)' },
        }}
      >
        {content}
      </ButtonBase>
    );
  }

  return <Box sx={cardSx}>{content}</Box>;
}
