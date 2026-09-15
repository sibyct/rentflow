import { useState } from 'react';
import { Outlet, useLocation } from 'react-router-dom';
import Box from '@mui/material/Box';
import useMediaQuery from '@mui/material/useMediaQuery';
import { Sidebar } from './Sidebar';
import { TopBar } from './TopBar';
import { PageHeader } from './PageHeader';
import { ChromeInteractivityProvider } from './ChromeInteractivityContext';
import { tokens } from '../tokens';
import { ErrorBoundary } from '@/shared/components';

/**
 * The authenticated app's shell: sidebar + top bar + page header own
 * their own layout; everything else renders through Outlet. Mirrors
 * AuthLayout's role for unauthenticated routes — this is the wrapper
 * for everything behind ProtectedRoute (see router.tsx).
 */
export function AppShell() {
  const isNarrow = useMediaQuery(`(max-width:${tokens.collapseBreakpoint - 1}px)`);
  // null = follow the auto-collapse breakpoint; once the user toggles
  // manually, their choice sticks regardless of viewport width.
  const [manualCollapsed, setManualCollapsed] = useState<boolean | null>(null);
  const collapsed = manualCollapsed ?? isNarrow;
  const location = useLocation();

  return (
    <ChromeInteractivityProvider>
      <Box sx={{ display: 'flex', minHeight: '100vh', bgcolor: tokens.slate[50] }}>
        <Sidebar collapsed={collapsed} onToggleCollapsed={() => setManualCollapsed(!collapsed)} />

        <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
          <TopBar />
          <Box component="main" sx={{ flex: 1, p: 3 }}>
            <PageHeader />
            {/* Scoped to just the routed page: if a page throws, the
                sidebar, top bar, and page header above stay live so the
                user can still navigate away — "Try again" re-renders
                just this page, not the whole app (see App.tsx's
                top-level boundary for that last resort). Keyed on the
                route so navigating to a different page after a crash
                starts that boundary fresh instead of staying stuck. */}
            <ErrorBoundary key={location.pathname} secondaryAction={{ label: 'Go to Dashboard', href: '/dashboard' }}>
              <Outlet />
            </ErrorBoundary>
          </Box>
        </Box>
      </Box>
    </ChromeInteractivityProvider>
  );
}
