import type { ReactNode } from 'react';
import { AppProviders } from './providers';
import { AppRouter } from './router';
import { AppLoadingScreen, ErrorBoundary } from '@/shared/components';
import { useSessionBootstrap } from '@/features/auth/hooks/useSessionBootstrap';

interface SessionBootstrapProps {
  children: ReactNode;
}

/**
 * On first load the access token in memory is gone (it was never
 * persisted), but the httpOnly refresh cookie may still be valid.
 * useSessionBootstrap silently exchanges it for a new access token
 * before rendering routes, so an authenticated user isn't bounced to
 * /login on every page reload.
 */
function SessionBootstrap({ children }: SessionBootstrapProps) {
  const { isPending } = useSessionBootstrap();

  if (isPending) {
    return <AppLoadingScreen />;
  }

  return <>{children}</>;
}

export function App() {
  return (
    // Catastrophic-failure boundary only: this sits above AppProviders,
    // so a crash here means the router/query client/theme themselves
    // never mounted — "Try again" (remount) is the only in-app recovery
    // available, so a full reload is offered too. Anything that fails
    // *after* the shell is up (a single page, a widget) should be
    // caught by a boundary further down instead (see AppShell), so
    // navigation and the rest of the app stay usable — this one is
    // deliberately the last resort, not the only line of defense.
    <ErrorBoundary secondaryAction={{ label: 'Reload page', onClick: () => window.location.reload() }}>
      <AppProviders>
        <SessionBootstrap>
          <AppRouter />
        </SessionBootstrap>
      </AppProviders>
    </ErrorBoundary>
  );
}
