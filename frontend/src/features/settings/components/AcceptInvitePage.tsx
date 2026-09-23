import { useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Chip from '@mui/material/Chip';
import IconButton from '@mui/material/IconButton';
import InputAdornment from '@mui/material/InputAdornment';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import CheckCircleOutlined from '@mui/icons-material/CheckCircleOutlined';
import ScheduleOutlined from '@mui/icons-material/ScheduleOutlined';
import Visibility from '@mui/icons-material/Visibility';
import VisibilityOff from '@mui/icons-material/VisibilityOff';
import { tokens } from '@/app/tokens';
import { BrandMark } from '@/shared/components';
import { acceptInviteSchema, type AcceptInviteFormValues } from '../schemas/acceptInviteSchema';
import { formatPropertyAccess } from '../utils';
import { ROLE_LABELS, type PropertyAccess, type StaffRole } from '../types';

type InviteVariant = 'invited' | 'expired';

// A stand-in for what a real "GET /invites/:token" would return.
const MOCK_INVITE: { name: string; email: string; role: StaffRole; propertyAccess: PropertyAccess; invitedBy: string } = {
  name: 'Tom Reyes',
  email: 'tom.reyes@harborpg.com',
  role: 'accountant',
  propertyAccess: { all: true, properties: [] },
  invitedBy: 'Alex Doran',
};

/**
 * The page a staff invite email links to. This is a front-end-only
 * prototype — see ../types.ts — so there is no real token or backend
 * call behind it. Both outcomes (a valid invite, and one that's expired)
 * are reachable here via the small "Preview as" toggle rather than two
 * separate routes, since without a backend there's no real token to
 * expire.
 */
export function AcceptInvitePage() {
  const [variant, setVariant] = useState<InviteVariant>('invited');
  const [accepted, setAccepted] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  const {
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<AcceptInviteFormValues>({
    resolver: zodResolver(acceptInviteSchema),
    mode: 'onTouched',
    defaultValues: { password: '', confirmPassword: '' },
  });

  return (
    <Box>
      <BrandMark sx={{ mb: 5, display: { xs: 'flex', md: 'none' } }} />

      <Paper
        variant="outlined"
        sx={{ display: 'flex', alignItems: 'center', gap: 1, p: 1, mb: 4, bgcolor: tokens.slate[50], borderStyle: 'dashed' }}
      >
        <Typography sx={{ fontSize: 11.5, color: tokens.slate[500], pl: 0.5, flex: 1 }}>Preview as</Typography>
        <ButtonGroup size="small">
          <Button
            variant={variant === 'invited' ? 'contained' : 'outlined'}
            onClick={() => {
              setVariant('invited');
              setAccepted(false);
            }}
            sx={variant !== 'invited' ? { borderColor: tokens.slate[300], color: tokens.slate[700] } : undefined}
          >
            Valid invite
          </Button>
          <Button
            variant={variant === 'expired' ? 'contained' : 'outlined'}
            onClick={() => setVariant('expired')}
            sx={variant !== 'expired' ? { borderColor: tokens.slate[300], color: tokens.slate[700] } : undefined}
          >
            Expired invite
          </Button>
        </ButtonGroup>
      </Paper>

      {accepted ? (
        <Stack spacing={2} sx={{ alignItems: 'flex-start' }}>
          <CheckCircleOutlined sx={{ fontSize: 36, color: tokens.brand.green }} />
          <Typography variant="h3" sx={{ fontSize: '1.5rem' }}>
            You&apos;re all set
          </Typography>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[500] }}>
            Your password has been set. In the real product this would sign you straight in.
          </Typography>
        </Stack>
      ) : variant === 'expired' ? (
        <Stack spacing={2} sx={{ alignItems: 'flex-start' }}>
          <Box sx={{ width: 48, height: 48, borderRadius: '50%', bgcolor: tokens.warningTint, color: tokens.warningInk, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <ScheduleOutlined />
          </Box>
          <Typography variant="h3" sx={{ fontSize: '1.5rem' }}>
            This invite has expired
          </Typography>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[500] }}>
            Invites to {MOCK_INVITE.email} expire 7 days after they&apos;re sent. Ask {MOCK_INVITE.invitedBy} to resend it from
            Settings &rsaquo; Users &amp; roles.
          </Typography>
          <Button variant="outlined" disabled sx={{ borderColor: tokens.slate[300] }}>
            Request a new invite
          </Button>
        </Stack>
      ) : (
        <Box>
          <Typography variant="h3" sx={{ fontWeight: 700, fontSize: '1.75rem' }}>
            Set your password
          </Typography>
          <Typography variant="body1" color="text.secondary" sx={{ mt: 1, mb: 3 }}>
            {MOCK_INVITE.invitedBy} invited you to join RentFlow.
          </Typography>

          <Paper variant="outlined" sx={{ p: 2, mb: 3 }}>
            <Stack spacing={1}>
              <Stack direction="row" sx={{ justifyContent: 'space-between' }}>
                <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>Name</Typography>
                <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{MOCK_INVITE.name}</Typography>
              </Stack>
              <Stack direction="row" sx={{ justifyContent: 'space-between' }}>
                <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>Email</Typography>
                <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{MOCK_INVITE.email}</Typography>
              </Stack>
              <Stack direction="row" sx={{ justifyContent: 'space-between', alignItems: 'center' }}>
                <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>Role</Typography>
                <Chip label={ROLE_LABELS[MOCK_INVITE.role]} size="small" color="info" sx={{ fontWeight: 600 }} />
              </Stack>
              <Stack direction="row" sx={{ justifyContent: 'space-between' }}>
                <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>Property access</Typography>
                <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{formatPropertyAccess(MOCK_INVITE.propertyAccess)}</Typography>
              </Stack>
            </Stack>
          </Paper>

          <Box component="form" onSubmit={handleSubmit(() => setAccepted(true))} noValidate>
            <Stack spacing={2.5}>
              <Controller
                name="password"
                control={control}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Password"
                    type={showPassword ? 'text' : 'password'}
                    fullWidth
                    error={Boolean(errors.password)}
                    helperText={errors.password?.message ?? 'At least 8 characters.'}
                    slotProps={{
                      input: {
                        endAdornment: (
                          <InputAdornment position="end">
                            <IconButton aria-label={showPassword ? 'Hide password' : 'Show password'} onClick={() => setShowPassword((v) => !v)} edge="end" size="small">
                              {showPassword ? <VisibilityOff fontSize="small" /> : <Visibility fontSize="small" />}
                            </IconButton>
                          </InputAdornment>
                        ),
                      },
                    }}
                  />
                )}
              />
              <Controller
                name="confirmPassword"
                control={control}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Confirm password"
                    type={showPassword ? 'text' : 'password'}
                    fullWidth
                    error={Boolean(errors.confirmPassword)}
                    helperText={errors.confirmPassword?.message}
                  />
                )}
              />

              <Alert severity="info" sx={{ fontSize: 12.5 }}>
                This is a design prototype — no account is actually created.
              </Alert>

              <Button type="submit" variant="contained" size="large" fullWidth>
                Accept invite &amp; set password
              </Button>
            </Stack>
          </Box>
        </Box>
      )}
    </Box>
  );
}
