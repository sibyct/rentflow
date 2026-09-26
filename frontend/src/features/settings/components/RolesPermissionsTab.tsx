import Box from '@mui/material/Box';
import Paper from '@mui/material/Paper';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import CheckCircleOutlined from '@mui/icons-material/CheckCircleOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';
import VisibilityOutlined from '@mui/icons-material/VisibilityOutlined';
import BlockOutlined from '@mui/icons-material/BlockOutlined';
import InfoOutlined from '@mui/icons-material/InfoOutlined';
import { tokens } from '@/app/tokens';
import { useStaff } from '../hooks/useStaffQueries';
import { PERMISSION_MATRIX, ROLE_DESCRIPTIONS, ROLE_LABELS, STAFF_ROLES, type PermissionLevel } from '../types';

const LEVEL_META: Record<PermissionLevel, { label: string; color: string; icon: typeof CheckCircleOutlined }> = {
  full: { label: 'Full', color: tokens.brand.green, icon: CheckCircleOutlined },
  edit: { label: 'Edit', color: tokens.azure[600], icon: EditOutlined },
  view: { label: 'View', color: tokens.slate[500], icon: VisibilityOutlined },
  none: { label: 'No access', color: tokens.slate[300], icon: BlockOutlined },
};

/** Read-only summary of what each role can do. Property scoping (the "Always all properties" / "Scoped to assigned properties" line on each role card) is enforced server-side on every property/unit/lease/work-order request. The module-level Full/Edit/View/No-access matrix below is still a reference only — no route checks it yet — so the note below the table is load-bearing, not boilerplate. */
export function RolesPermissionsTab() {
  const { data: users = [] } = useStaff();
  const activeCount = (role: (typeof STAFF_ROLES)[number]) => users.filter((u) => u.role === role && u.status !== 'deactivated').length;

  return (
    <Box>
      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)', md: 'repeat(4, 1fr)' }, mb: 3 }}>
        {STAFF_ROLES.map((role) => (
          <Paper key={role} variant="outlined" sx={{ p: 2.5 }}>
            <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', mb: 1 }}>
              <Typography sx={{ fontSize: 14.5, fontWeight: 700 }}>{ROLE_LABELS[role]}</Typography>
              <Box sx={{ fontSize: 12, fontWeight: 700, color: tokens.slate[600], bgcolor: tokens.slate[100], borderRadius: '999px', px: 1, py: 0.2 }}>
                {activeCount(role)}
              </Box>
            </Box>
            <Typography sx={{ fontSize: 12.5, color: tokens.slate[500], mb: 1.5, lineHeight: 1.5, minHeight: 36 }}>
              {ROLE_DESCRIPTIONS[role]}
            </Typography>
            <Typography sx={{ fontSize: 11.5, color: tokens.slate[400] }}>
              {role === 'admin' ? 'Always all properties' : 'Scoped to assigned properties'}
            </Typography>
          </Paper>
        ))}
      </Box>

      <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
        <TableContainer sx={{ overflowX: 'auto' }}>
          <Table size="small" sx={{ minWidth: 720 }}>
            <TableHead>
              <TableRow>
                <TableCell>Module</TableCell>
                {STAFF_ROLES.map((role) => (
                  <TableCell key={role}>{ROLE_LABELS[role]}</TableCell>
                ))}
              </TableRow>
            </TableHead>
            <TableBody>
              {PERMISSION_MATRIX.map((row) => (
                <TableRow key={row.module} hover>
                  <TableCell>
                    <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{row.module}</Typography>
                  </TableCell>
                  {STAFF_ROLES.map((role) => {
                    const level = row.levels[role];
                    const meta = LEVEL_META[level];
                    const Icon = meta.icon;
                    return (
                      <TableCell key={role}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, color: meta.color }}>
                          <Icon sx={{ fontSize: 16 }} />
                          <Typography sx={{ fontSize: 13, color: level === 'none' ? tokens.slate[400] : tokens.slate[700] }}>{meta.label}</Typography>
                        </Box>
                      </TableCell>
                    );
                  })}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>

      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, mt: 1.5 }}>
        <InfoOutlined sx={{ fontSize: 15, color: tokens.slate[400] }} />
        <Tooltip title="Property scoping is enforced server-side: a staff member scoped to specific properties gets a 404 on any property/unit/lease/work order outside that scope, on every request. Which module actions (Full/Edit/View/No access) a role gets is not yet checked by the API — this matrix is a reference for that part only.">
          <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>Property scoping is enforced server-side. This module matrix is still a reference.</Typography>
        </Tooltip>
      </Box>
    </Box>
  );
}
