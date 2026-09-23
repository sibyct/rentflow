import { Link as RouterLink, Outlet, useLocation } from 'react-router-dom';
import Box from '@mui/material/Box';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import { tokens } from '@/app/tokens';

const ACCOUNTING_TABS: { label: string; path: string }[] = [
  { label: 'Dashboard', path: '/accounting' },
  { label: 'Rent Roll', path: '/accounting/rent-roll' },
  { label: 'Expenses', path: '/accounting/expenses' },
  { label: 'Charges', path: '/accounting/charges' },
];

/** The tab whose path is the longest prefix of the current route ('/accounting' only matches itself exactly). */
function activeTabPath(pathname: string): string | false {
  const match = ACCOUNTING_TABS.filter(
    (t) => pathname === t.path || (t.path !== '/accounting' && pathname.startsWith(`${t.path}/`)),
  ).sort((a, b) => b.path.length - a.path.length)[0];
  return match?.path ?? false;
}

/**
 * Shared frame for every /accounting/* page: route-driven tabs (each tab
 * is a real link, so the URL, back button and deep links all work) above
 * an <Outlet/>. Tabs scroll rather than wrap on narrow screens so the
 * page itself never scrolls horizontally.
 */
export function AccountingLayout() {
  const { pathname } = useLocation();

  return (
    <Box>
      <Tabs
        value={activeTabPath(pathname)}
        variant="scrollable"
        scrollButtons="auto"
        aria-label="Accounting sections"
        sx={{ mb: 3, borderBottom: `1px solid ${tokens.slate[200]}` }}
      >
        {ACCOUNTING_TABS.map((t) => (
          <Tab
            key={t.path}
            value={t.path}
            label={t.label}
            component={RouterLink}
            to={t.path}
            sx={{ textTransform: 'none', fontWeight: 600 }}
          />
        ))}
      </Tabs>
      <Outlet />
    </Box>
  );
}
