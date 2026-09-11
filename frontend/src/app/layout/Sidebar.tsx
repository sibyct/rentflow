import { Link as RouterLink, useLocation } from 'react-router-dom';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import MenuOpenOutlined from '@mui/icons-material/MenuOpenOutlined';
import MenuOutlined from '@mui/icons-material/MenuOutlined';
import { BrandMark } from '@/shared/components';
import { tokens } from '../tokens';
import { navGroups, settingsNavItem, findNavItemByPath, type NavItem } from './navConfig';

interface SidebarProps {
  collapsed: boolean;
  onToggleCollapsed: () => void;
}

export function Sidebar({ collapsed, onToggleCollapsed }: SidebarProps) {
  const location = useLocation();
  const activeId = findNavItemByPath(location.pathname)?.item.id;

  return (
    <Box
      component="nav"
      sx={{
        width: collapsed ? tokens.navWidthCollapsed : tokens.navWidthExpanded,
        flexShrink: 0,
        height: '100vh',
        position: 'sticky',
        top: 0,
        display: 'flex',
        flexDirection: 'column',
        bgcolor: 'common.white',
        borderRight: `1px solid ${tokens.slate[200]}`,
        overflow: 'hidden',
        transition: 'width 150ms ease',
      }}
    >
      <SidebarHeader collapsed={collapsed} onToggleCollapsed={onToggleCollapsed} />

      <Box sx={{ flex: 1, overflowY: 'auto', overflowX: 'hidden', px: collapsed ? 1 : 1.5, py: 1.5 }}>
        <Stack spacing={2}>
          {navGroups.map((group) => (
            <Box key={group.id}>
              {group.label &&
                (collapsed ? (
                  <Box sx={{ height: '1px', bgcolor: tokens.slate[200], my: 1 }} />
                ) : (
                  <Typography
                    sx={{
                      fontFamily: tokens.fontMono,
                      fontSize: 10,
                      fontWeight: 500,
                      letterSpacing: '0.08em',
                      textTransform: 'uppercase',
                      color: tokens.slate[400],
                      px: 1,
                      mb: 0.5,
                    }}
                  >
                    {group.label}
                  </Typography>
                ))}
              <Stack spacing={0.5}>
                {group.items.map((item) => (
                  <NavRow key={item.id} item={item} active={item.id === activeId} collapsed={collapsed} />
                ))}
              </Stack>
            </Box>
          ))}
        </Stack>
      </Box>

      {!collapsed && (
        <Box
          sx={{
            mx: 1.5,
            mb: 1.5,
            p: 1.5,
            borderRadius: `${tokens.radiusCard}px`,
            bgcolor: tokens.warningTint,
            color: tokens.warningInk,
          }}
        >
          <Typography sx={{ fontSize: 13, fontWeight: 600 }}>3 leases expire in 30 days</Typography>
          <Typography
            component={RouterLink}
            to="/leases"
            sx={{ fontSize: 13, fontWeight: 600, textDecoration: 'underline', color: 'inherit', display: 'block', mt: 0.25 }}
          >
            Review renewals
          </Typography>
        </Box>
      )}

      <Box sx={{ borderTop: `1px solid ${tokens.slate[100]}`, px: collapsed ? 1 : 1.5, py: 1.5 }}>
        <NavRow item={settingsNavItem} active={settingsNavItem.id === activeId} collapsed={collapsed} />
      </Box>
    </Box>
  );
}

function SidebarHeader({ collapsed, onToggleCollapsed }: SidebarProps) {
  if (collapsed) {
    return (
      <Stack sx={{ alignItems: 'center', py: 1 }}>
        <Box sx={{ height: 40, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <BrandMark size={28} nameHidden />
        </Box>
        <Box sx={{ height: 40, width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Tooltip title="Expand sidebar" placement="right">
            <IconButton size="small" onClick={onToggleCollapsed} aria-label="Expand sidebar">
              <MenuOutlined fontSize="small" />
            </IconButton>
          </Tooltip>
        </Box>
      </Stack>
    );
  }

  return (
    <Stack
      direction="row"
      sx={{ alignItems: 'center', justifyContent: 'space-between', height: tokens.topBarHeight, px: 1.5 }}
    >
      <BrandMark size={28} nameHidden/>
      <Tooltip title="Collapse sidebar" placement="right">
        <IconButton size="small" onClick={onToggleCollapsed} aria-label="Collapse sidebar">
          <MenuOpenOutlined fontSize="small" />
        </IconButton>
      </Tooltip>
    </Stack>
  );
}

interface NavRowProps {
  item: NavItem;
  active: boolean;
  collapsed: boolean;
}

function NavRow({ item, active, collapsed }: NavRowProps) {
  const Icon = item.icon;

  const row = (
    <ButtonBase
      component={item.disabled ? 'div' : RouterLink}
      to={item.disabled ? undefined : item.path}
      disabled={item.disabled}
      sx={{
        width: '100%',
        height: tokens.navItemHeight,
        borderRadius: `${tokens.radiusControl}px`,
        px: 1,
        justifyContent: collapsed ? 'center' : 'flex-start',
        gap: 1.25,
        color: item.disabled ? tokens.slate[300] : active ? tokens.azure[700] : tokens.slate[700],
        bgcolor: active ? tokens.azure[50] : 'transparent',
        fontWeight: active ? 600 : 500,
        fontSize: 14,
        cursor: item.disabled ? 'not-allowed' : 'pointer',
        boxShadow: active ? `inset 2px 0 0 ${tokens.azure[500]}` : 'none',
        '&:hover': item.disabled ? undefined : { bgcolor: active ? tokens.azure[50] : tokens.slate[50] },
        '&.Mui-focusVisible': { boxShadow: `${active ? `inset 2px 0 0 ${tokens.azure[500]}, ` : ''}${tokens.focusRing}` },
      }}
    >
      <Icon sx={{ fontSize: 20, flexShrink: 0 }} />
      {!collapsed && (
        <>
          <Typography sx={{ flex: 1, textAlign: 'left', fontSize: 'inherit', fontWeight: 'inherit', color: 'inherit' }}>
            {item.label}
          </Typography>
          {item.badge && <BadgePill count={item.badge.count} kind={item.badge.kind} />}
        </>
      )}
    </ButtonBase>
  );

  if (!collapsed) return row;

  return (
    <Tooltip title={item.label} placement="right">
      <span>{row}</span>
    </Tooltip>
  );
}

function BadgePill({ count, kind }: { count: number; kind: 'urgent' | 'info' }) {
  return (
    <Box
      sx={{
        minWidth: 20,
        height: 20,
        px: 0.75,
        borderRadius: 10,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: 11,
        fontWeight: 600,
        fontFamily: tokens.fontMono,
        bgcolor: kind === 'urgent' ? tokens.error : tokens.slate[100],
        color: kind === 'urgent' ? '#FFFFFF' : tokens.slate[600],
      }}
    >
      {count}
    </Box>
  );
}
