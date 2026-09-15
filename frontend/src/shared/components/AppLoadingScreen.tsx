import Box from '@mui/material/Box';
import LinearProgress from '@mui/material/LinearProgress';
import Stack from '@mui/material/Stack';
import { tokens } from '@/app/tokens';
import { BrandMark } from './BrandMark';

/**
 * Full-viewport branded screen shown while the app itself is still
 * getting ready — before there's a session, a route, or anything else
 * to render (see App.tsx's SessionBootstrap, which shows this during
 * the silent /auth/refresh on first load). Not for in-page loading
 * states once the app is up — use LoadingSpinner or QueryState there,
 * since this replaces the *entire* screen.
 */
export function AppLoadingScreen() {
  return (
    <Box
      role="status"
      aria-live="polite"
      aria-label="Loading RentFlow"
      sx={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        bgcolor: tokens.slate[50],
      }}
    >
      <Stack sx={{ alignItems: 'center', gap: 3 }}>
        <BrandMark size={40} />
        <LinearProgress
          sx={{
            width: 160,
            height: 4,
            borderRadius: 999,
            bgcolor: tokens.slate[200],
            '& .MuiLinearProgress-bar': { borderRadius: 999, bgcolor: tokens.brand.green },
          }}
        />
      </Stack>
    </Box>
  );
}
