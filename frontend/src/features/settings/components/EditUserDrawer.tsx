import { useEffect, useState } from 'react';
import { ApiError } from '@/api/client';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import CircularProgress from '@mui/material/CircularProgress';
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
import LockResetOutlined from '@mui/icons-material/LockResetOutlined';
import ReplayOutlined from '@mui/icons-material/ReplayOutlined';
import InfoOutlined from '@mui/icons-material/InfoOutlined';
import { tokens } from '@/app/tokens';
import { useProperties } from '@/features/properties/hooks/usePropertiesQueries';
import { FormSection, StatusChip } from '@/shared/components';
import { relativeTime } from '@/shared/lib/relativeTime';
import { useDeactivateStaff, useReactivateStaff, useResetStaffPassword, useUpdateStaff } from '../hooks/useStaffQueries';
import { initials } from '../utils';
import { ROLE_LABELS, STAFF_ROLES, USER_STATUS_LABELS, USER_STATUS_TONE, type StaffRole, type StaffUser } from '../types';

interface Toast {
  message: string;
  severity: 'success' | 'error';
}

interface EditUserDrawerProps {
  user: StaffUser | null;
  onClose: () => void;
  isLastActiveAdmin: (userId: string) => boolean;
  onToast: (toast: Toast) => void;
}

function errorMessage(err: unknown): string {
  return err instanceof ApiError ? err.message : 'Something went wrong. Please try again.';
}

/**
 * Read + edit in place, like WorkOrderDrawer elsewhere in this app —
 * role and property scope are editable; identity fields
 * are read-only. The last-active-Admin safeguard disables both the role
 * select and the Deactivate action, with an inline explanation, rather
 * than letting an account lock itself out of Settings. The account's
 * root owner (isAccountOwner) can't be managed here at all — the server
 * rejects it outright (see StaffService.requireManageableStaff).
 */
export function EditUserDrawer({ user, onClose, isLastActiveAdmin, onToast }: EditUserDrawerProps) {
  const updateUser = useUpdateStaff();
  const deactivate = useDeactivateStaff();
  const reactivate = useReactivateStaff();
  const resetPassword = useResetStaffPassword();
  const { data: propertiesResult } = useProperties({ limit: 500, offset: 0 });
  const properties = propertiesResult?.properties ?? [];

  const [role, setRole] = useState<StaffRole>('property_manager');
  const [accessAll, setAccessAll] = useState(true);
  const [propertyIds, setPropertyIds] = useState<string[]>([]);
  const [confirmDeactivate, setConfirmDeactivate] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  useEffect(() => {
    if (user) {
      setRole(user.role);
      setAccessAll(user.propertyAccess.all);
      setPropertyIds(user.propertyAccess.propertyIds);
      setSubmitError(null);
    }
  }, [user]);

  if (!user) return null;

  const locked = user.isAccountOwner || isLastActiveAdmin(user.id);

  function toggleProperty(id: string) {
    setPropertyIds((prev) => (prev.includes(id) ? prev.filter((p) => p !== id) : [...prev, id]));
  }

  function handleSave() {
    if (!user) return;
    setSubmitError(null);
    updateUser.mutate(
      { id: user.id, input: { role, propertyAccess: role === 'admin' ? { all: true, propertyIds: [] } : { all: accessAll, propertyIds } } },
      {
        onSuccess: () => {
          onToast({ message: `${user.name}’s access was updated`, severity: 'success' });
          onClose();
        },
        onError: (err) => setSubmitError(errorMessage(err)),
      },
    );
  }

  function handleDeactivate() {
    if (!user) return;
    deactivate.mutate(user.id, {
      onSuccess: () => {
        setConfirmDeactivate(false);
        onToast({ message: `${user.name} was deactivated`, severity: 'success' });
        onClose();
      },
      onError: (err) => {
        setConfirmDeactivate(false);
        onToast({ message: errorMessage(err), severity: 'error' });
      },
    });
  }

  function handleReactivate() {
    if (!user) return;
    reactivate.mutate(user.id, {
      onSuccess: () => {
        onToast({ message: `${user.name} was reactivated`, severity: 'success' });
        onClose();
      },
      onError: (err) => onToast({ message: errorMessage(err), severity: 'error' }),
    });
  }

  function handleResetPassword() {
    if (!user) return;
    resetPassword.mutate(user.id, {
      onSuccess: () => onToast({ message: `Password reset link sent to ${user.email}`, severity: 'success' }),
      onError: (err) => onToast({ message: errorMessage(err), severity: 'error' }),
    });
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
            {submitError && (
              <Alert severity="error" onClose={() => setSubmitError(null)}>
                {submitError}
              </Alert>
            )}

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

            {user.isAccountOwner ? (
              <Alert severity="info" icon={<InfoOutlined fontSize="small" />}>
                This is the account's owner — role and access can't be changed here.
              </Alert>
            ) : (
              isLastActiveAdmin(user.id) && (
                <Alert severity="warning" icon={<InfoOutlined fontSize="small" />}>
                  This is the only active Admin on the account. Promote another user to Admin before changing this role or
                  deactivating this account.
                </Alert>
              )
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
                      {properties.map((p) => (
                        <FormControlLabel
                          key={p.id}
                          control={<Checkbox size="small" disabled={locked} checked={propertyIds.includes(p.id)} onChange={() => toggleProperty(p.id)} />}
                          label={<Typography sx={{ fontSize: 13 }}>{p.name}</Typography>}
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
          {user.isAccountOwner ? (
            <span />
          ) : user.status === 'active' ? (
            <Stack direction="row" spacing={0.5}>
              <Button
                variant="text"
                startIcon={<LockResetOutlined fontSize="small" />}
                disabled={resetPassword.isPending}
                onClick={handleResetPassword}
                sx={{ color: tokens.slate[700] }}
              >
                Reset password
              </Button>
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
            </Stack>
          ) : user.status === 'deactivated' ? (
            <Button variant="text" startIcon={<ReplayOutlined fontSize="small" />} disabled={reactivate.isPending} onClick={handleReactivate}>
              Reactivate
            </Button>
          ) : (
            <span />
          )}

          <Stack direction="row" spacing={1.5}>
            <Button variant="text" onClick={onClose} sx={{ color: tokens.slate[600] }}>
              Cancel
            </Button>
            <Button
              variant="contained"
              onClick={handleSave}
              disabled={locked || updateUser.isPending}
              startIcon={updateUser.isPending ? <CircularProgress size={15} color="inherit" /> : undefined}
            >
              {updateUser.isPending ? 'Saving…' : 'Save changes'}
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
          <Button variant="contained" color="error" disabled={deactivate.isPending} onClick={handleDeactivate}>
            Deactivate
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
