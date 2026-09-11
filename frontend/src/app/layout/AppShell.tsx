import { useState } from 'react';
import { Outlet } from 'react-router-dom';
import Box from '@mui/material/Box';
import useMediaQuery from '@mui/material/useMediaQuery';
import { Sidebar } from './Sidebar';
import { TopBar } from './TopBar';
import { PageHeader } from './PageHeader';
import { ChromeInteractivityProvider } from './ChromeInteractivityContext';
import { tokens } from '../tokens';

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

  return (
    <ChromeInteractivityProvider>
      <Box sx={{ display: 'flex', minHeight: '100vh', bgcolor: tokens.slate[50] }}>
        <Sidebar collapsed={collapsed} onToggleCollapsed={() => setManualCollapsed(!collapsed)} />

        <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
          <TopBar />
          <Box component="main" sx={{ flex: 1, p: 3 }}>
            <PageHeader />
            <Outlet />
          </Box>
        </Box>
      </Box>
    </ChromeInteractivityProvider>
  );
}
