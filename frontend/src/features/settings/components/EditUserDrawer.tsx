import { useEffect, useState } from 'react';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import Divider from '@mui/material/Divider';
import Drawer from '@mui/material/Drawer';
import FormControlLabel from '@mui/material/FormControlLabel';
import IconButton from '@mui/material/IconButton';
import MenuItem from '@mui/material/MenuItem';
import Radio from '@mui/material/Radio';
import RadioGroup from '@mui/material/RadioGroup';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import CloseOutlined from '@mui/icons-material/CloseOutlined';
import BlockOutlined from '@mui/icons-material/BlockOutlined';
import ReplayOutlined from '@mui/icons-material/ReplayOutlined';
import InfoOutlined from '@mui/icons-material/InfoOutlined';
import { tokens } from '@/app/tokens';
import { FormSection, StatusChip } from '@/shared/components';
import { relativeTime } from '@/shared/lib/relativeTime';
import { useUsersRolesStore } from '../store/useUsersRolesStore';
import { initials } from '../utils';
import {
  MOCK_PROPERTIES,
  ROLE_LABELS,
  STAFF_ROLES,
  USER_STATUS_LABELS,
  USER_STATUS_TONE,
  type PropertyAccess,
  type StaffRole,
  type StaffUser,
} from '../types';

interface EditUserDrawerProps {
  user: StaffUser | null;
  onClose: () => void;
}

/**
 * Read + edit in place, like WorkOrderDrawer/UnitDetailDrawer elsewhere
 * in this app — role and property scope are editable; identity fields
 * are read-only. The last-active-Admin safeguard disables both the role
 * select and the Deactivate action, with an inline explanation, rather
 * than letting an account lock itself out of Settings.
 */
export function EditUserDrawer({ user, onClose }: EditUserDrawerProps) {
  const updateUser = useUsersRolesStore((s) => s.updateUser);
  const deactivate = useUsersRolesStore((s) => s.deactivate);
  const reactivate = useUsersRolesStore((s) => s.reactivate);
  const isLastActiveAdmin = useUsersRolesStore((s) => s.isLastActiveAdmin);

  const [role, setRole] = useState<StaffRole>('property_manager');
  const [accessAll, setAccessAll] = useState(true);
  const [properties, setProperties] = useState<string[]>([]);
  const [confirmDeactivate, setConfirmDeactivate] = useState(false);

  useEffect(() => {
    if (user) {
      setRole(user.role);
      setAccessAll(user.propertyAccess.all);
      setProperties(user.propertyAccess.properties);
    }
  }, [user]);

  if (!user) return null;

  const locked = isLastActiveAdmin(user.id);

  function toggleProperty(name: string) {
    setProperties((prev) => (prev.includes(name) ? prev.filter((p) => p !== name) : [...prev, name]));
  }

  function handleSave() {
    if (!user) return;
    const propertyAccess: PropertyAccess = role === 'admin' ? { all: true, properties: [] } : { all: accessAll, properties };
    updateUser(user.id, { role, propertyAccess });
    onClose();
  }

  return (
    <>
      <Drawer
        anchor="right"
        open={Boolean(user)}
        onClose={onClose}
        slotProps={{ paper: { sx: { width: { xs: '100%', sm: 480 }, display: 'flex', flexDirection: 'column' } } }}
      >
        <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', p: 2.5, pb: 1.5 }}>
          <Typography variant="h5">Edit user</Typography>
          <IconButton onClick={onClose} aria-label="Close">
            <CloseOutlined />
          </IconButton>
        </Stack>
        <Divider />

        <Box sx={{ flex: 1, overflowY: 'auto', p: 2.5 }}>
          <Stack spacing={3}>
            <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center' }}>
              <Box
                sx={{
                  width: 44,
                  height: 44,
                  borderRadius: '50%',
                  bgcolor: tokens.slate[700],
                  color: 'common.white',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 15,
                  fontWeight: 700,
                  flexShrink: 0,
                }}
              >
                {initials(user.name)}
              </Box>
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <Typography sx={{ fontSize: 15, fontWeight: 700 }}>{user.name}</Typography>
                <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>{user.email}</Typography>
              </Box>
              <StatusChip label={USER_STATUS_LABELS[user.status]} tone={USER_STATUS_TONE[user.status]} />
            </Stack>

            <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>
              Last login: {user.lastLoginAt ? relativeTime(user.lastLoginAt) : 'Never'}
            </Typography>

            {locked && (
              <Alert severity="warning" icon={<InfoOutlined fontSize="small" />}>
                This is the only active Admin on the account. Promote another user to Admin before changing this role or
                deactivating this account.
              </Alert>
            )}

            <FormSection title="Role">
              <TextField select value={role} disabled={locked} onChange={(e) => setRole(e.target.value as StaffRole)}>
                {STAFF_ROLES.map((r) => (
                  <MenuItem key={r} value={r}>
                    {ROLE_LABELS[r]}
                  </MenuItem>
                ))}
              </TextField>
            </FormSection>

            <FormSection title="Property access">
              {role === 'admin' ? (
                <Typography sx={{ fontSize: 13, color: tokens.slate[500] }}>
                  Admins always have access to every property.
                </Typography>
              ) : (
                <>
                  <RadioGroup value={accessAll ? 'all' : 'specific'} onChange={(e) => setAccessAll(e.target.value === 'all')}>
                    <FormControlLabel value="all" control={<Radio size="small" disabled={locked} />} label={<Typography sx={{ fontSize: 13.5 }}>All properties</Typography>} />
                    <FormControlLabel value="specific" control={<Radio size="small" disabled={locked} />} label={<Typography sx={{ fontSize: 13.5 }}>Specific properties</Typography>} />
                  </RadioGroup>
                  {!accessAll && (
                    <Box sx={{ mt: 1, ml: 4, display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)' }}>
                      {MOCK_PROPERTIES.map((p) => (
                        <FormControlLabel
                          key={p}
                          control={<Checkbox size="small" disabled={locked} checked={properties.includes(p)} onChange={() => toggleProperty(p)} />}
                          label={<Typography sx={{ fontSize: 13 }}>{p}</Typography>}
                        />
                      ))}
                    </Box>
                  )}
                </>
              )}
            </FormSection>
          </Stack>
        </Box>

        <Divider />
        <Stack direction="row" sx={{ p: 2, justifyContent: 'space-between', alignItems: 'center' }}>
          {user.status === 'active' ? (
            <Tooltip title={locked ? "This is the only active Admin on the account." : ''}>
              <span>
                <Button
                  variant="text"
                  color="error"
                  startIcon={<BlockOutlined fontSize="small" />}
                  disabled={locked}
                  onClick={() => setConfirmDeactivate(true)}
                >
                  Deactivate
                </Button>
              </span>
            </Tooltip>
          ) : user.status === 'deactivated' ? (
            <Button
              variant="text"
              startIcon={<ReplayOutlined fontSize="small" />}
              onClick={() => {
                reactivate(user.id);
                onClose();
              }}
            >
              Reactivate
            </Button>
          ) : (
            <span />
          )}

          <Stack direction="row" spacing={1.5}>
            <Button variant="text" onClick={onClose} sx={{ color: tokens.slate[600] }}>
              Cancel
            </Button>
            <Button variant="contained" onClick={handleSave}>
              Save changes
            </Button>
          </Stack>
        </Stack>
      </Drawer>

      <Dialog open={confirmDeactivate} onClose={() => setConfirmDeactivate(false)}>
        <DialogTitle>Deactivate {user.name}?</DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>
            {user.name} will lose access immediately. You can reactivate this account at any time.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 0 }}>
          <Button variant="text" onClick={() => setConfirmDeactivate(false)} sx={{ color: tokens.slate[600] }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color="error"
            onClick={() => {
              deactivate(user.id);
              setConfirmDeactivate(false);
              onClose();
            }}
          >
            Deactivate
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
