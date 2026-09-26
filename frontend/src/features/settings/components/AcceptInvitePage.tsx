import { useState } from 'react';
import { Link as RouterLink, useSearchParams } from 'react-router-dom';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import IconButton from '@mui/material/IconButton';
import InputAdornment from '@mui/material/InputAdornment';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import CheckCircleOutlined from '@mui/icons-material/CheckCircleOutlined';
import ErrorOutlineOutlined from '@mui/icons-material/ErrorOutlineOutlined';
import ScheduleOutlined from '@mui/icons-material/ScheduleOutlined';
import Visibility from '@mui/icons-material/Visibility';
import VisibilityOff from '@mui/icons-material/VisibilityOff';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import { BrandMark, LoadingSpinner } from '@/shared/components';
import { staffApi } from '../api/staffApi';
import { useInviteLookup } from '../hooks/useStaffQueries';
import { acceptInviteSchema, type AcceptInviteFormValues } from '../schemas/acceptInviteSchema';
import { formatPropertyAccess } from '../utils';
import { ROLE_LABELS } from '../types';

function errorMessage(err: unknown): string {
  return err instanceof ApiError ? err.message : 'Something went wrong. Please try again.';
}

/** The page a staff invite email links to (?token=...). */
export function AcceptInvitePage() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') ?? '';

  const { data: invite, isLoading, isError } = useInviteLookup(token);
  const [accepted, setAccepted] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const {
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<AcceptInviteFormValues>({
    resolver: zodResolver(acceptInviteSchema),
    mode: 'onTouched',
    defaultValues: { password: '', confirmPassword: '' },
  });

  async function onSubmit(values: AcceptInviteFormValues) {
    setSubmitError(null);
    setSubmitting(true);
    try {
      await staffApi.acceptInvite(token, values.password);
      setAccepted(true);
    } catch (err) {
      setSubmitError(errorMessage(err));
    } finally {
      setSubmitting(false);
    }
  }

  if (!token || isError) {
    return (
      <Box>
        <BrandMark sx={{ mb: 5, display: { xs: 'flex', md: 'none' } }} />
        <Stack spacing={2} sx={{ alignItems: 'flex-start' }}>
          <Box sx={{ width: 48, height: 48, borderRadius: '50%', bgcolor: 'rgba(194,38,47,0.1)', color: tokens.error, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <ErrorOutlineOutlined />
          </Box>
          <Typography variant="h3" sx={{ fontSize: '1.5rem' }}>
            This invite link isn&apos;t valid
          </Typography>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[500] }}>
            It may have already been used, or the link is incomplete. Ask whoever invited you to resend it from
            Settings &rsaquo; Users &amp; roles.
          </Typography>
        </Stack>
      </Box>
    );
  }

  if (isLoading) {
    return (
      <Box sx={{ py: 8 }}>
        <LoadingSpinner label="Checking your invite…" />
      </Box>
    );
  }

  if (accepted) {
    return (
      <Box>
        <BrandMark sx={{ mb: 5, display: { xs: 'flex', md: 'none' } }} />
        <Stack spacing={2} sx={{ alignItems: 'flex-start' }}>
          <CheckCircleOutlined sx={{ fontSize: 36, color: tokens.brand.green }} />
          <Typography variant="h3" sx={{ fontSize: '1.5rem' }}>
            You&apos;re all set
          </Typography>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[500] }}>Your password has been set.</Typography>
          <Button component={RouterLink} to="/login" variant="contained">
            Sign in
          </Button>
        </Stack>
      </Box>
    );
  }

  if (invite?.expired) {
    return (
      <Box>
        <BrandMark sx={{ mb: 5, display: { xs: 'flex', md: 'none' } }} />
        <Stack spacing={2} sx={{ alignItems: 'flex-start' }}>
          <Box sx={{ width: 48, height: 48, borderRadius: '50%', bgcolor: tokens.warningTint, color: tokens.warningInk, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <ScheduleOutlined />
          </Box>
          <Typography variant="h3" sx={{ fontSize: '1.5rem' }}>
            This invite has expired
          </Typography>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[500] }}>
            Invites to {invite.email} expire 7 days after they&apos;re sent. Ask {invite.invitedByName || 'your account admin'} to
            resend it from Settings &rsaquo; Users &amp; roles.
          </Typography>
        </Stack>
      </Box>
    );
  }

  if (!invite) return null;

  return (
    <Box>
      <BrandMark sx={{ mb: 5, display: { xs: 'flex', md: 'none' } }} />
      <Box>
        <Typography variant="h3" sx={{ fontWeight: 700, fontSize: '1.75rem' }}>
          Set your password
        </Typography>
        <Typography variant="body1" color="text.secondary" sx={{ mt: 1, mb: 3 }}>
          {invite.invitedByName || 'You'} invited you to join RentFlow.
        </Typography>

        <Paper variant="outlined" sx={{ p: 2, mb: 3 }}>
          <Stack spacing={1}>
            <Stack direction="row" sx={{ justifyContent: 'space-between' }}>
              <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>Name</Typography>
              <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{invite.name}</Typography>
            </Stack>
            <Stack direction="row" sx={{ justifyContent: 'space-between' }}>
              <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>Email</Typography>
              <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{invite.email}</Typography>
            </Stack>
            <Stack direction="row" sx={{ justifyContent: 'space-between', alignItems: 'center' }}>
              <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>Role</Typography>
              <Chip label={ROLE_LABELS[invite.role]} size="small" color="info" sx={{ fontWeight: 600 }} />
            </Stack>
            <Stack direction="row" sx={{ justifyContent: 'space-between' }}>
              <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>Property access</Typography>
              <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{formatPropertyAccess(invite.propertyAccess)}</Typography>
            </Stack>
          </Stack>
        </Paper>

        <Box component="form" onSubmit={handleSubmit(onSubmit)} noValidate>
          <Stack spacing={2.5}>
            {submitError && (
              <Alert severity="error" onClose={() => setSubmitError(null)}>
                {submitError}
              </Alert>
            )}
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

            <Button
              type="submit"
              variant="contained"
              size="large"
              fullWidth
              disabled={submitting}
              startIcon={submitting ? <CircularProgress size={16} color="inherit" /> : undefined}
            >
              {submitting ? 'Setting password…' : 'Accept invite & set password'}
            </Button>
          </Stack>
        </Box>
      </Box>
    </Box>
  );
}
