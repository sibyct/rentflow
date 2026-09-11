import { useState, type MouseEvent } from 'react';
import Badge from '@mui/material/Badge';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import Divider from '@mui/material/Divider';
import IconButton from '@mui/material/IconButton';
import InputAdornment from '@mui/material/InputAdornment';
import Popover from '@mui/material/Popover';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import KeyboardArrowDownOutlined from '@mui/icons-material/KeyboardArrowDownOutlined';
import NotificationsOutlined from '@mui/icons-material/NotificationsOutlined';
import SearchOutlined from '@mui/icons-material/SearchOutlined';
import { useAuth, useLogout } from '@/features/auth';
import { tokens } from '../tokens';

// Static placeholder content: there's no notifications backend yet. Kept
// here rather than invented server-side so it's obvious this is demo
// content for the shell, not real data.
const DEMO_NOTIFICATIONS = [
  { id: 1, title: 'Lease renewal due', meta: 'Unit 4B · expires in 12 days' },
  { id: 2, title: 'New maintenance request', meta: 'Unit 2A · submitted 3h ago' },
  { id: 3, title: 'Payment received', meta: 'Unit 7C · $1,450' },
];

export function TopBar() {
  const [openPopover, setOpenPopover] = useState<'notifications' | 'account' | null>(null);
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
  const { user } = useAuth();
  const logout = useLogout();

  const handleTrigger = (key: 'notifications' | 'account') => (e: MouseEvent<HTMLElement>) => {
    if (openPopover === key) {
      setOpenPopover(null);
      setAnchorEl(null);
      return;
    }
    setAnchorEl(e.currentTarget);
    setOpenPopover(key);
  };

  const closePopover = () => {
    setOpenPopover(null);
    setAnchorEl(null);
  };

  const initials = (user?.email ?? '??').slice(0, 2).toUpperCase();

  return (
    <Box
      component="header"
      sx={{
        height: tokens.topBarHeight,
        position: 'sticky',
        top: 0,
        zIndex: (t) => t.zIndex.appBar,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        px: 2.5,
        bgcolor: 'common.white',
        borderBottom: `1px solid ${tokens.slate[200]}`,
      }}
    >
      <TextField
        placeholder="Search units, residents, work orders"
        size="small"
        sx={{
          width: '100%',
          maxWidth: 420,
          '& .MuiOutlinedInput-root': {
            height: 34,
            bgcolor: tokens.slate[50],
            borderRadius: `${tokens.radiusControl}px`,
            '& fieldset': { border: 'none' },
            '&:hover': { bgcolor: tokens.slate[50] },
            '&.Mui-focused': {
              bgcolor: 'common.white',
              boxShadow: tokens.focusRing,
            },
          },
        }}
        slotProps={{
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <SearchOutlined sx={{ fontSize: 18, color: tokens.slate[400] }} />
              </InputAdornment>
            ),
            endAdornment: (
              <InputAdornment position="end">
                <Box
                  sx={{
                    px: 0.75,
                    py: 0.25,
                    borderRadius: 1,
                    border: `1px solid ${tokens.slate[300]}`,
                    fontFamily: tokens.fontMono,
                    fontSize: 11,
                    color: tokens.slate[500],
                    bgcolor: 'common.white',
                  }}
                >
                  ⌘K
                </Box>
              </InputAdornment>
            ),
          },
        }}
      />

      <Stack direction="row" sx={{ alignItems: 'center', gap: 1.5 }}>
        <IconButton onClick={handleTrigger('notifications')} aria-label="Notifications">
          <Badge
            variant="dot"
            color="error"
            overlap="circular"
            anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
          >
            <NotificationsOutlined sx={{ fontSize: 20, color: tokens.slate[600] }} />
          </Badge>
        </IconButton>

        <Divider orientation="vertical" flexItem sx={{ borderColor: tokens.slate[200], my: 1 }} />

        <ButtonBase
          onClick={handleTrigger('account')}
          sx={{ display: 'flex', alignItems: 'center', gap: 0.75, p: 0.5, borderRadius: 1 }}
          aria-label="Account menu"
        >
          <Box
            sx={{
              width: 32,
              height: 32,
              borderRadius: '50%',
              bgcolor: tokens.slate[100],
              color: tokens.slate[700],
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 12,
              fontWeight: 600,
            }}
          >
            {initials}
          </Box>
          <KeyboardArrowDownOutlined sx={{ fontSize: 18, color: tokens.slate[500] }} />
        </ButtonBase>
      </Stack>

      <Popover
        open={openPopover === 'notifications'}
        anchorEl={anchorEl}
        onClose={closePopover}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        slotProps={{ paper: { sx: { width: 300, mt: 1, borderRadius: `${tokens.radiusCard}px`, boxShadow: tokens.shadowPopover } } }}
      >
        <Box sx={{ px: 2, py: 1.5 }}>
          <Typography sx={{ fontWeight: 700, fontSize: 14 }}>Notifications</Typography>
        </Box>
        {DEMO_NOTIFICATIONS.map((n) => (
          <Box
            key={n.id}
            sx={{
              px: 2,
              py: 1.25,
              borderTop: `1px solid ${tokens.slate[100]}`,
              '&:hover': { bgcolor: tokens.slate[50] },
            }}
          >
            <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{n.title}</Typography>
            <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>{n.meta}</Typography>
          </Box>
        ))}
      </Popover>

      <Popover
        open={openPopover === 'account'}
        anchorEl={anchorEl}
        onClose={closePopover}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        slotProps={{ paper: { sx: { width: 300, mt: 1, borderRadius: `${tokens.radiusCard}px`, boxShadow: tokens.shadowPopover } } }}
      >
        <Box sx={{ px: 2, py: 1.5 }}>
          <Typography sx={{ fontWeight: 700, fontSize: 14 }}>{user?.email}</Typography>
          <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>{user?.role}</Typography>
        </Box>
        <Box
          onClick={() => {
            closePopover();
            logout.mutate();
          }}
          sx={{
            px: 2,
            py: 1.25,
            borderTop: `1px solid ${tokens.slate[100]}`,
            cursor: 'pointer',
            '&:hover': { bgcolor: tokens.slate[50] },
          }}
        >
          <Typography sx={{ fontSize: 13, fontWeight: 600 }}>Sign out</Typography>
        </Box>
      </Popover>
    </Box>
  );
}
