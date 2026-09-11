import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import type { SxProps, Theme } from '@mui/material/styles';
import { tokens } from '@/app/tokens';
import { RentFlowMark } from './RentFlowMark';

interface BrandMarkProps {
  size?: number;
  light?: boolean;
  /** Renders just the logo mark, no wordmark — for the collapsed nav rail. */
  nameHidden?: boolean;
  sx?: SxProps<Theme>;
}

/** App logo mark + wordmark, reused across auth pages and the app shell's sidebar. */
export function BrandMark({ size = 32, light = false, nameHidden = false, sx }: BrandMarkProps) {
  return (
    <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center', ...sx }}>
      {nameHidden ? (
        // Standalone icon-only usage (sidebar header, collapsed rail) needs
        // a tile behind the mark — the bare linework reads as a stray
        // squiggle without one. Matches the favicon artwork's tile treatment.
        <Box
          sx={{
            width: size,
            height: size,
            flexShrink: 0,
            borderRadius: `${size * 0.3}px`,
            bgcolor: tokens.brand.ink,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <RentFlowMark size={size * 0.62} variant="white" />
        </Box>
      ) : (
        <>
          <RentFlowMark size={size} variant={light ? 'white' : 'color'} />
          <Typography
            sx={{
              fontWeight: 700,
              fontSize: size * 0.6,
              letterSpacing: '-0.02em',
              color: light ? 'common.white' : 'text.primary',
            }}
          >
            Rent
            <Typography
              component="span"
              sx={{ fontWeight: 'inherit', fontSize: 'inherit', letterSpacing: 'inherit', color: light ? tokens.brand.mint : tokens.brand.green }}
            >
              Flow
            </Typography>
          </Typography>
        </>
      )}
    </Stack>
  );
}
