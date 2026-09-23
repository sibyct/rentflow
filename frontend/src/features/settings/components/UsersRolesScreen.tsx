import { useMemo, useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import InputAdornment from '@mui/material/InputAdornment';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import ArrowBackOutlined from '@mui/icons-material/ArrowBackOutlined';
import PersonAddAlt1Outlined from '@mui/icons-material/PersonAddAlt1Outlined';
import SearchOutlined from '@mui/icons-material/SearchOutlined';
import { tokens } from '@/app/tokens';
import { useUsersRolesStore } from '../store/useUsersRolesStore';
import { AuditLogTab } from './AuditLogTab';
import { EditUserDrawer } from './EditUserDrawer';
import { InviteUserDialog } from './InviteUserDialog';
import { RolesPermissionsTab } from './RolesPermissionsTab';
import { UsersTable } from './UsersTable';
import { ROLE_LABELS, STAFF_ROLES, USER_STATUS_LABELS, type StaffUser, type StaffRole, type UserStatus } from '../types';

type TabKey = 'users' | 'roles' | 'audit';
const STATUSES: UserStatus[] = ['active', 'invited', 'invite_expired', 'deactivated'];

/**
 * Front-end prototype of the Users & Roles settings section — see
 * ../types.ts for why: this codebase has no multi-user/staff backend
 * yet. All state lives in useUsersRolesStore and resets on reload.
 */
export function UsersRolesScreen() {
  const users = useUsersRolesStore((s) => s.users);
  const toast = useUsersRolesStore((s) => s.toast);
  const dismissToast = useUsersRolesStore((s) => s.dismissToast);
  const resendInvite = useUsersRolesStore((s) => s.resendInvite);
  const deactivate = useUsersRolesStore((s) => s.deactivate);
  const reactivate = useUsersRolesStore((s) => s.reactivate);
  const isLastActiveAdmin = useUsersRolesStore((s) => s.isLastActiveAdmin);

  const [tab, setTab] = useState<TabKey>('users');
  const [search, setSearch] = useState('');
  const [roleFilter, setRoleFilter] = useState<StaffRole | ''>('');
  const [statusFilter, setStatusFilter] = useState<UserStatus | ''>('');
  const [inviteOpen, setInviteOpen] = useState(false);
  const [editUser, setEditUser] = useState<StaffUser | null>(null);

  const filteredUsers = useMemo(() => {
    const q = search.trim().toLowerCase();
    return users.filter((u) => {
      if (q && !u.name.toLowerCase().includes(q) && !u.email.toLowerCase().includes(q)) return false;
      if (roleFilter && u.role !== roleFilter) return false;
      if (statusFilter && u.status !== statusFilter) return false;
      return true;
    });
  }, [users, search, roleFilter, statusFilter]);

  return (
    <Box>
      <Link
        component={RouterLink}
        to="/settings"
        sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, fontSize: 13, fontWeight: 600, color: tokens.slate[600], mb: 1, '&:hover': { color: tokens.azure[600] } }}
      >
        <ArrowBackOutlined sx={{ fontSize: 15 }} />
        Settings
      </Link>

      <Stack direction="row" sx={{ alignItems: 'flex-start', justifyContent: 'space-between', flexWrap: 'wrap', gap: 2, mb: 3 }}>
        <Box>
          <Typography variant="h3">Users & roles</Typography>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[500], mt: 0.5 }}>
            Invite staff, set what each role can do, and limit people to the properties they work on.
          </Typography>
        </Box>
        <Button variant="contained" startIcon={<PersonAddAlt1Outlined />} onClick={() => setInviteOpen(true)}>
          Invite user
        </Button>
      </Stack>

      <Tabs value={tab} onChange={(_e, v: TabKey) => setTab(v)} sx={{ borderBottom: `1px solid ${tokens.slate[200]}`, mb: 3 }}>
        <Tab value="users" label={`Users (${users.length})`} sx={{ textTransform: 'none', fontWeight: 600 }} />
        <Tab value="roles" label="Roles & permissions" sx={{ textTransform: 'none', fontWeight: 600 }} />
        <Tab value="audit" label="Audit log" sx={{ textTransform: 'none', fontWeight: 600 }} />
      </Tabs>

      {tab === 'users' && (
        <Box>
          <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center', flexWrap: 'wrap', mb: 2, rowGap: 1.25 }}>
            <TextField
              size="small"
              placeholder="Search name or email"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              sx={{ flex: '1 1 200px', maxWidth: 300 }}
              slotProps={{
                input: {
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchOutlined sx={{ fontSize: 18, color: tokens.slate[400] }} />
                    </InputAdornment>
                  ),
                },
              }}
            />
            <TextField select size="small" slotProps={{ select: { displayEmpty: true } }} value={roleFilter} onChange={(e) => setRoleFilter(e.target.value as StaffRole | '')} sx={{ minWidth: 170 }}>
              <MenuItem value="">Role: All</MenuItem>
              {STAFF_ROLES.map((r) => (
                <MenuItem key={r} value={r}>
                  {ROLE_LABELS[r]}
                </MenuItem>
              ))}
            </TextField>
            <TextField select size="small" slotProps={{ select: { displayEmpty: true } }} value={statusFilter} onChange={(e) => setStatusFilter(e.target.value as UserStatus | '')} sx={{ minWidth: 160 }}>
              <MenuItem value="">Status: All</MenuItem>
              {STATUSES.map((s) => (
                <MenuItem key={s} value={s}>
                  {USER_STATUS_LABELS[s]}
                </MenuItem>
              ))}
            </TextField>
            <Box sx={{ flex: 1 }} />
            <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>
              {filteredUsers.length} of {users.length} users
            </Typography>
          </Stack>

          <UsersTable
            users={filteredUsers}
            totalUnfiltered={users.length}
            onOpenEdit={setEditUser}
            onResendInvite={(u) => resendInvite(u.id)}
            onDeactivate={(u) => deactivate(u.id)}
            onReactivate={(u) => reactivate(u.id)}
            isLastActiveAdmin={isLastActiveAdmin}
          />
        </Box>
      )}

      {tab === 'roles' && <RolesPermissionsTab />}
      {tab === 'audit' && <AuditLogTab />}

      <InviteUserDialog open={inviteOpen} onClose={() => setInviteOpen(false)} />
      <EditUserDrawer user={editUser} onClose={() => setEditUser(null)} />

      <Snackbar open={Boolean(toast)} autoHideDuration={3200} onClose={dismissToast} anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}>
        {toast ? (
          <Alert onClose={dismissToast} severity={toast.severity} variant="filled" sx={{ borderRadius: 999 }}>
            {toast.message}
          </Alert>
        ) : undefined}
      </Snackbar>
    </Box>
  );
}
