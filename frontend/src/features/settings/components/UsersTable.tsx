import { useState } from 'react';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import MoreVertOutlined from '@mui/icons-material/MoreVertOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';
import MailOutlined from '@mui/icons-material/MailOutlined';
import LockResetOutlined from '@mui/icons-material/LockResetOutlined';
import BlockOutlined from '@mui/icons-material/BlockOutlined';
import ReplayOutlined from '@mui/icons-material/ReplayOutlined';
import PersonSearchOutlined from '@mui/icons-material/PersonSearchOutlined';
import { tokens } from '@/app/tokens';
import { EmptyState, StatusChip } from '@/shared/components';
import { relativeTime } from '@/shared/lib/relativeTime';
import { formatPropertyAccess, initials, inviteExpiryLabel, propertyAccessSubtext } from '../utils';
import { ROLE_LABELS, USER_STATUS_LABELS, USER_STATUS_TONE, type StaffUser } from '../types';

interface UsersTableProps {
  users: StaffUser[];
  totalUnfiltered: number;
  onOpenEdit: (user: StaffUser) => void;
  onResendInvite: (user: StaffUser) => void;
  onResetPassword: (user: StaffUser) => void;
  onDeactivate: (user: StaffUser) => void;
  onReactivate: (user: StaffUser) => void;
  isLastActiveAdmin: (userId: string) => boolean;
}

export function UsersTable({ users, totalUnfiltered, onOpenEdit, onResendInvite, onResetPassword, onDeactivate, onReactivate, isLastActiveAdmin }: UsersTableProps) {
  const [menu, setMenu] = useState<{ user: StaffUser; anchor: HTMLElement } | null>(null);

  const closeMenu = () => setMenu(null);

  return (
    <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
      <TableContainer sx={{ overflowX: 'auto' }}>
        <Table size="small" sx={{ minWidth: 900 }}>
          {users.length > 0 && (
            <TableHead>
              <TableRow>
                <TableCell>User</TableCell>
                <TableCell>Role</TableCell>
                <TableCell>Property access</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Last login</TableCell>
                <TableCell padding="checkbox" />
              </TableRow>
            </TableHead>
          )}
          <TableBody>
            {users.length === 0 && (
              <TableRow>
                <TableCell colSpan={6}>
                  {totalUnfiltered === 0 ? (
                    <EmptyState icon={PersonSearchOutlined} title="No staff yet" description="Invite your first teammate to get started." />
                  ) : (
                    <EmptyState icon={PersonSearchOutlined} title="No users match your filters" description="Try a different search term or filter." />
                  )}
                </TableCell>
              </TableRow>
            )}

            {users.map((user) => {
              return (
                <TableRow key={user.id} hover>
                  <TableCell onClick={() => onOpenEdit(user)} sx={{ cursor: 'pointer' }}>
                    <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center' }}>
                      <Box
                        sx={{
                          width: 32,
                          height: 32,
                          borderRadius: '50%',
                          bgcolor: tokens.slate[700],
                          color: 'common.white',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          fontSize: 12,
                          fontWeight: 700,
                          flexShrink: 0,
                        }}
                      >
                        {initials(user.name)}
                      </Box>
                      <Box sx={{ minWidth: 0 }}>
                        <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center' }}>
                          <Typography sx={{ fontSize: 13.5, fontWeight: 600, color: tokens.slate[900] }}>{user.name}</Typography>
                          {user.isCurrentUser && (
                            <Box sx={{ fontSize: 10.5, fontWeight: 700, color: tokens.slate[500], bgcolor: tokens.slate[100], borderRadius: '4px', px: 0.6, py: 0.1 }}>
                              You
                            </Box>
                          )}
                        </Stack>
                        <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>{user.email}</Typography>
                      </Box>
                    </Stack>
                  </TableCell>

                  <TableCell>
                    <Typography sx={{ fontSize: 13 }}>{ROLE_LABELS[user.role]}</Typography>
                  </TableCell>

                  <TableCell>
                    <Typography sx={{ fontSize: 13 }}>{formatPropertyAccess(user.propertyAccess)}</Typography>
                    {propertyAccessSubtext(user.propertyAccess) && (
                      <Typography sx={{ fontSize: 11.5, color: tokens.slate[500] }}>{propertyAccessSubtext(user.propertyAccess)}</Typography>
                    )}
                  </TableCell>

                  <TableCell>
                    <StatusChip label={USER_STATUS_LABELS[user.status]} tone={USER_STATUS_TONE[user.status]} />
                    {user.inviteExpiresAt && (user.status === 'invited' || user.status === 'invite_expired') && (
                      <Typography sx={{ fontSize: 11, color: tokens.slate[500], mt: 0.4 }}>{inviteExpiryLabel(user.inviteExpiresAt)}</Typography>
                    )}
                  </TableCell>

                  <TableCell>
                    <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>
                      {user.lastLoginAt ? relativeTime(user.lastLoginAt) : 'Never'}
                    </Typography>
                  </TableCell>

                  <TableCell padding="checkbox" onClick={(e) => e.stopPropagation()}>
                    <Tooltip title="Actions">
                      <IconButton size="small" onClick={(e) => setMenu({ user, anchor: e.currentTarget })} aria-label={`Actions for ${user.name}`}>
                        <MoreVertOutlined fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </TableContainer>

      <Menu anchorEl={menu?.anchor} open={Boolean(menu)} onClose={closeMenu}>
        {menu && [
          <MenuItem
            key="edit"
            onClick={() => {
              onOpenEdit(menu.user);
              closeMenu();
            }}
          >
            <EditOutlined fontSize="small" sx={{ mr: 1.25, color: tokens.slate[500] }} />
            Edit
          </MenuItem>,

          (menu.user.status === 'invited' || menu.user.status === 'invite_expired') && !menu.user.isAccountOwner && (
            <MenuItem
              key="resend"
              onClick={() => {
                onResendInvite(menu.user);
                closeMenu();
              }}
            >
              <MailOutlined fontSize="small" sx={{ mr: 1.25, color: tokens.slate[500] }} />
              Resend invite
            </MenuItem>
          ),

          menu.user.status === 'active' && !menu.user.isAccountOwner && (
            <MenuItem
              key="reset-password"
              onClick={() => {
                onResetPassword(menu.user);
                closeMenu();
              }}
            >
              <LockResetOutlined fontSize="small" sx={{ mr: 1.25, color: tokens.slate[500] }} />
              Reset password
            </MenuItem>
          ),

          menu.user.status === 'active' && !menu.user.isAccountOwner && (
            <Tooltip key="deactivate" title={isLastActiveAdmin(menu.user.id) ? "This is the account's only active Admin." : ''} placement="left">
              <span>
                <MenuItem
                  disabled={isLastActiveAdmin(menu.user.id)}
                  onClick={() => {
                    onDeactivate(menu.user);
                    closeMenu();
                  }}
                  sx={{ color: isLastActiveAdmin(menu.user.id) ? undefined : 'error.main' }}
                >
                  <BlockOutlined fontSize="small" sx={{ mr: 1.25 }} />
                  Deactivate
                </MenuItem>
              </span>
            </Tooltip>
          ),

          menu.user.status === 'deactivated' && !menu.user.isAccountOwner && (
            <MenuItem
              key="reactivate"
              onClick={() => {
                onReactivate(menu.user);
                closeMenu();
              }}
            >
              <ReplayOutlined fontSize="small" sx={{ mr: 1.25, color: tokens.slate[500] }} />
              Reactivate
            </MenuItem>
          ),
        ]}
      </Menu>
    </Paper>
  );
}
