import LockClockOutlined from '@mui/icons-material/LockClockOutlined';
import BlockOutlined from '@mui/icons-material/BlockOutlined';
import WrongLocationOutlined from '@mui/icons-material/WrongLocationOutlined';
import StorageOutlined from '@mui/icons-material/StorageOutlined';
import WifiOffOutlined from '@mui/icons-material/WifiOffOutlined';
import type { ErrorStateProps } from './ErrorState';

/**
 * Icon/title/message/tone/dimChrome for the errors this app actually
 * surfaces. Actions are left out — "Log in" needs the app's login route,
 * "Reload" needs `window.location.reload`, etc. — so call sites spread a
 * preset and add `primaryAction`/`secondaryAction` themselves:
 *
 *   <ErrorState {...ERROR_STATE_PRESETS.sessionExpired}
 *     primaryAction={{ label: 'Log in', href: '/login' }} />
 */
export const ERROR_STATE_PRESETS = {
  sessionExpired: {
    icon: LockClockOutlined,
    title: 'Your session has expired',
    message: "For your security, we signed you out after a period of inactivity. Log in again to pick up where you left off.",
    tone: 'neutral',
    dimChrome: true,
  },
  forbidden: {
    icon: BlockOutlined,
    title: "You don't have access to this page",
    message: 'Your account doesn’t have permission to view this. If you think that’s wrong, ask an admin to check your role.',
    tone: 'neutral',
    dimChrome: false,
  },
  notFound: {
    icon: WrongLocationOutlined,
    title: "We can't find that page",
    message: "The page you're looking for may have been moved or no longer exists. Check the URL, or head back to the dashboard.",
    tone: 'neutral',
    dimChrome: false,
  },
  serverError: {
    icon: StorageOutlined,
    title: 'Something went wrong on our end',
    message: "This page failed to load because of a server error. It's not something you did — try again in a moment, or let us know if it keeps happening.",
    tone: 'neutral',
    dimChrome: false,
  },
  offline: {
    icon: WifiOffOutlined,
    title: "You're offline",
    message: 'We can’t reach the server right now. Check your connection — this page will keep waiting and load automatically once you’re back online.',
    tone: 'neutral',
    dimChrome: true,
  },
} as const satisfies Record<string, Pick<ErrorStateProps, 'icon' | 'title' | 'message' | 'tone' | 'dimChrome'>>;
