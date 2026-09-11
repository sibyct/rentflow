import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { tokens } from '@/app/tokens';

interface PlaceholderPageProps {
  label: string;
}

/** Stand-in content for a nav destination that doesn't have a real page yet. */
export function PlaceholderPage({ label }: PlaceholderPageProps) {
  return (
    <Box
      sx={{
        minHeight: 320,
        border: `1px dashed ${tokens.slate[300]}`,
        borderRadius: `${tokens.radiusCard}px`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        textAlign: 'center',
        px: 3,
      }}
    >
      <Typography sx={{ color: tokens.slate[500] }}>
        Page content for{' '}
        <Box component="span" sx={{ fontWeight: 700, color: tokens.slate[700] }}>
          {label}
        </Box>{' '}
        renders here — the shell owns only the nav, top bar and page header.
      </Typography>
    </Box>
  );
}
