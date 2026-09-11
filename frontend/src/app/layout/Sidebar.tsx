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
import { useChromeDimmed } from './ChromeInteractivityContext';

interface SidebarProps {
  collapsed: boolean;
  onToggleCollapsed: () => void;
}

export function Sidebar({ collapsed, onToggleCollapsed }: SidebarProps) {
  const location = useLocation();
  const activeId = findNavItemByPath(location.pathname)?.item.id;
  const dimmed = useChromeDimmed();

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

      <Box
        sx={{
          flex: 1,
          overflowY: 'auto',
          overflowX: 'hidden',
          px: collapsed ? 1 : 1.5,
          py: 2,
          opacity: dimmed ? 0.5 : 1,
          pointerEvents: dimmed ? 'none' : 'auto',
          transition: 'opacity 150ms ease',
        }}
      >
        <Stack spacing={2.5}>
          {navGroups.map((group) => (
            <Box key={group.id}>
              {group.label &&
                (collapsed ? (
                  <Box sx={{ height: '1px', bgcolor: tokens.slate[200], my: 1 }} />
                ) : (
                  <Typography
                    sx={{
                      fontSize: 11,
                      fontWeight: 600,
                      letterSpacing: '0.04em',
                      textTransform: 'uppercase',
                      color: tokens.slate[300],
                      px: 1,
                      mb: 1,
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

      <Box
        sx={{
          borderTop: `1px solid ${tokens.slate[100]}`,
          px: collapsed ? 1 : 1.5,
          py: 1.25,
          opacity: dimmed ? 0.5 : 1,
          pointerEvents: dimmed ? 'none' : 'auto',
          transition: 'opacity 150ms ease',
        }}
      >
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
      <BrandMark size={26} />
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
  // Matches the containing scroll box's own padding (px: collapsed ? 1 : 1.5
  // in Sidebar's render), so the accent bar's negative offset lands it
  // exactly on the sidebar's left edge instead of the row's own edge.
  const railOffset = collapsed ? -8 : -12;

  const row = (
    <Box sx={{ position: 'relative' }}>
      {active && (
        <Box
          sx={{
            position: 'absolute',
            left: railOffset,
            top: 6,
            bottom: 6,
            width: 3,
            borderRadius: '0 3px 3px 0',
            bgcolor: tokens.azure[500],
          }}
        />
      )}
      <ButtonBase
        component={item.disabled ? 'div' : RouterLink}
        to={item.disabled ? undefined : item.path}
        disabled={item.disabled}
        sx={{
          width: '100%',
          height: tokens.navItemHeight,
          borderRadius: '10px',
          px: 1,
          justifyContent: collapsed ? 'center' : 'flex-start',
          gap: 1.25,
          color: item.disabled ? tokens.slate[300] : active ? tokens.azure[700] : tokens.slate[600],
          fontWeight: active ? 600 : 500,
          fontSize: 14,
          cursor: item.disabled ? 'not-allowed' : 'pointer',
          '&:hover': item.disabled ? undefined : { bgcolor: tokens.slate[50], color: active ? tokens.azure[700] : tokens.slate[900] },
          '&.Mui-focusVisible': { boxShadow: tokens.focusRing },
        }}
      >
        <Box
          sx={{
            width: 32,
            height: 32,
            flexShrink: 0,
            borderRadius: '8px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            bgcolor: active ? tokens.azure[500] : 'transparent',
            color: active ? '#FFFFFF' : 'inherit',
            transition: 'background-color 100ms ease',
          }}
        >
          <Icon sx={{ fontSize: 20 }} />
        </Box>
        {!collapsed && (
          <>
            <Typography sx={{ flex: 1, textAlign: 'left', fontSize: 'inherit', fontWeight: 'inherit', color: 'inherit' }}>
              {item.label}
            </Typography>
            {item.badge && <BadgePill count={item.badge.count} kind={item.badge.kind} />}
          </>
        )}
      </ButtonBase>
    </Box>
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
