import type { ComponentType } from 'react';
import type { SvgIconProps } from '@mui/material/SvgIcon';
import DashboardOutlined from '@mui/icons-material/DashboardOutlined';
import HomeOutlined from '@mui/icons-material/HomeOutlined';
import ApartmentOutlined from '@mui/icons-material/ApartmentOutlined';
import DescriptionOutlined from '@mui/icons-material/DescriptionOutlined';
import BuildOutlined from '@mui/icons-material/BuildOutlined';
import StorefrontOutlined from '@mui/icons-material/StorefrontOutlined';
import CalculateOutlined from '@mui/icons-material/CalculateOutlined';
import BarChartOutlined from '@mui/icons-material/BarChartOutlined';
import SettingsOutlined from '@mui/icons-material/SettingsOutlined';

export interface NavItem {
  id: string;
  label: string;
  icon: ComponentType<SvgIconProps>;
  /** Omitted only for disabled items, which aren't routable yet. */
  path?: string;
  badge?: { count: number; kind: 'urgent' | 'info' };
  disabled?: boolean;
}

export interface NavGroup {
  id: string;
  /** Omitted for the ungrouped top section (just Dashboard). */
  label?: string;
  items: NavItem[];
}

// Single source of truth for the sidebar, the page header's breadcrumb/H1
// (derived from whichever item matches the current route), and routing —
// see router.tsx, which generates a <Route> per item that has a path.
export const navGroups: NavGroup[] = [
  {
    id: 'top',
    items: [{ id: 'dashboard', label: 'Dashboard', icon: DashboardOutlined, path: '/dashboard' }],
  },
  {
    id: 'portfolio',
    label: 'Portfolio',
    items: [
      { id: 'properties', label: 'Properties', icon: HomeOutlined, path: '/properties' },
      { id: 'units', label: 'Units', icon: ApartmentOutlined, path: '/units' },
      { id: 'leases', label: 'Leases', icon: DescriptionOutlined, path: '/leases', badge: { count: 3, kind: 'info' } },
    ],
  },
  {
    id: 'operations',
    label: 'Operations',
    items: [
      {
        id: 'maintenance',
        label: 'Maintenance',
        icon: BuildOutlined,
        path: '/maintenance',
        badge: { count: 7, kind: 'urgent' },
      },
      { id: 'vendors', label: 'Vendors', icon: StorefrontOutlined, path: '/vendors' },
    ],
  },
  {
    id: 'finance',
    label: 'Finance',
    items: [
      { id: 'accounting', label: 'Accounting', icon: CalculateOutlined, disabled: true },
      { id: 'reports', label: 'Reports', icon: BarChartOutlined, disabled: true },
    ],
  },
];

export const settingsNavItem: NavItem = {
  id: 'settings',
  label: 'Settings',
  icon: SettingsOutlined,
  path: '/settings',
};

export const allNavItems: NavItem[] = [...navGroups.flatMap((g) => g.items), settingsNavItem];

/**
 * Looks up the nav item (and its group label, for the breadcrumb)
 * matching a route path. Falls back to a prefix match so nested routes
 * like /properties/new or /properties/:id still highlight "Properties"
 * and show its breadcrumb/title, without needing an entry of their own.
 */
export function findNavItemByPath(pathname: string): { item: NavItem; groupLabel: string } | undefined {
  const exact = allNavItemsWithGroup().find(({ item }) => item.path === pathname);
  if (exact) return exact;

  return allNavItemsWithGroup()
    .filter(({ item }) => item.path && pathname.startsWith(`${item.path}/`))
    .sort((a, b) => (b.item.path?.length ?? 0) - (a.item.path?.length ?? 0))[0];
}

function allNavItemsWithGroup(): { item: NavItem; groupLabel: string }[] {
  const fromGroups = navGroups.flatMap((group) =>
    group.items.map((item) => ({ item, groupLabel: group.label ?? 'Home' })),
  );
  return [...fromGroups, { item: settingsNavItem, groupLabel: 'Home' }];
}
