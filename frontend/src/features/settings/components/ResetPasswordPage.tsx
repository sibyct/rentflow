import { useState } from 'react';
import { Link as RouterLink, useSearchParams } from 'react-router-dom';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import IconButton from '@mui/material/IconButton';
import InputAdornment from '@mui/material/InputAdornment';
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
import { usePasswordResetLookup } from '../hooks/useStaffQueries';
import { acceptInviteSchema, type AcceptInviteFormValues } from '../schemas/acceptInviteSchema';

function errorMessage(err: unknown): string {
  return err instanceof ApiError ? err.message : 'Something went wrong. Please try again.';
}

/** The page an admin-triggered password-reset email links to (?token=...). */
export function ResetPasswordPage() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') ?? '';

  const { data: reset, isLoading, isError } = usePasswordResetLookup(token);
  const [done, setDone] = useState(false);
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
      await staffApi.confirmPasswordReset(token, values.password);
      setDone(true);
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
            This reset link isn&apos;t valid
          </Typography>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[500] }}>
            It may have already been used, or the link is incomplete. Ask your account admin to reset your password
            again from Settings &rsaquo; Users &amp; roles.
          </Typography>
        </Stack>
      </Box>
    );
  }

  if (isLoading) {
    return (
      <Box sx={{ py: 8 }}>
        <LoadingSpinner label="Checking your reset link…" />
      </Box>
    );
  }

  if (done) {
    return (
      <Box>
        <BrandMark sx={{ mb: 5, display: { xs: 'flex', md: 'none' } }} />
        <Stack spacing={2} sx={{ alignItems: 'flex-start' }}>
          <CheckCircleOutlined sx={{ fontSize: 36, color: tokens.brand.green }} />
          <Typography variant="h3" sx={{ fontSize: '1.5rem' }}>
            Your password has been reset
          </Typography>
          <Button component={RouterLink} to="/login" variant="contained">
            Sign in
          </Button>
        </Stack>
      </Box>
    );
  }

  if (reset?.expired) {
    return (
      <Box>
        <BrandMark sx={{ mb: 5, display: { xs: 'flex', md: 'none' } }} />
        <Stack spacing={2} sx={{ alignItems: 'flex-start' }}>
          <Box sx={{ width: 48, height: 48, borderRadius: '50%', bgcolor: tokens.warningTint, color: tokens.warningInk, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <ScheduleOutlined />
          </Box>
          <Typography variant="h3" sx={{ fontSize: '1.5rem' }}>
            This reset link has expired
          </Typography>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[500] }}>
            Reset links expire 24 hours after they&apos;re sent. Ask your account admin to reset your password again
            from Settings &rsaquo; Users &amp; roles.
          </Typography>
        </Stack>
      </Box>
    );
  }

  if (!reset) return null;

  return (
    <Box>
      <BrandMark sx={{ mb: 5, display: { xs: 'flex', md: 'none' } }} />
      <Box>
        <Typography variant="h3" sx={{ fontWeight: 700, fontSize: '1.75rem' }}>
          Set a new password
        </Typography>
        <Typography variant="body1" color="text.secondary" sx={{ mt: 1, mb: 3 }}>
          For {reset.email}
        </Typography>

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
                  label="New password"
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
                  label="Confirm new password"
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
              {submitting ? 'Saving…' : 'Set new password'}
            </Button>
          </Stack>
        </Box>
      </Box>
    </Box>
  );
}
