import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import type { SxProps, Theme } from '@mui/material/styles';

interface BrandMarkProps {
  size?: number;
  light?: boolean;
  /** Renders just the logo square, no wordmark — for the collapsed nav rail. */
  nameHidden?: boolean;
  sx?: SxProps<Theme>;
}

/** App logo mark + name, reused across auth pages and the app shell's sidebar. */
export function BrandMark({ size = 32, light = false, nameHidden = false, sx }: BrandMarkProps) {
  return (
    <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center', ...sx }}>
      <Box
        sx={{
          width: size,
          height: size,
          flexShrink: 0,
          borderRadius: 1.5,
          bgcolor: 'primary.main',
          color: 'primary.contrastText',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontWeight: 800,
          fontSize: size * 0.5,
        }}
      >
        P
      </Box>
      {!nameHidden && (
        <Typography variant="h6" sx={{ fontWeight: 700, color: light ? 'common.white' : 'text.primary' }}>
          PropertyManagement
        </Typography>
      )}
    </Stack>
  );
}
