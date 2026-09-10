import { useEffect, useState, type ReactNode } from 'react';
import Box from '@mui/material/Box';
import { AppProviders } from './providers';
import { AppRouter } from './router';
import { ErrorBoundary, LoadingSpinner } from '@/shared/components';
import { authApi } from '@/features/auth/api/authApi';
import { useAuthStore } from '@/features/auth/store/authStore';

interface SessionBootstrapProps {
  children: ReactNode;
}

/**
 * On first load the access token in memory is gone (it was never
 * persisted), but the httpOnly refresh cookie may still be valid. This
 * silently exchanges it for a new access token before rendering routes,
 * so an authenticated user isn't bounced to /login on every page reload.
 */
function SessionBootstrap({ children }: SessionBootstrapProps) {
  const [ready, setReady] = useState(false);
  const setAuth = useAuthStore((s) => s.setAuth);

  useEffect(() => {
    let cancelled = false;

    authApi
      .refresh()
      .then((result) => {
        if (!cancelled) setAuth(result.accessToken, result.user);
      })
      .catch(() => {
        // No valid session cookie yet — starting logged out is expected.
      })
      .finally(() => {
        if (!cancelled) setReady(true);
      });

    return () => {
      cancelled = true;
    };
  }, [setAuth]);

  if (!ready) {
    return (
      <Box sx={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <LoadingSpinner label="Loading session…" />
      </Box>
    );
  }

  return <>{children}</>;
}

export function App() {
  return (
    <ErrorBoundary>
      <AppProviders>
        <SessionBootstrap>
          <AppRouter />
        </SessionBootstrap>
      </AppProviders>
    </ErrorBoundary>
  );
}
