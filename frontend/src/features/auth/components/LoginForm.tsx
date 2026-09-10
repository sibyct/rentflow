import { useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { Controller, useForm } from 'react-hook-form';
import { Link as RouterLink } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import IconButton from '@mui/material/IconButton';
import InputAdornment from '@mui/material/InputAdornment';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import Visibility from '@mui/icons-material/Visibility';
import VisibilityOff from '@mui/icons-material/VisibilityOff';
import { useLogin } from '../hooks/useAuthMutations';
import { loginSchema, type LoginFormValues } from '../schemas/authSchema';

interface LoginFormProps {
  onSuccess?: () => void;
}

export function LoginForm({ onSuccess }: LoginFormProps) {
  const login = useLogin();
  const [showPassword, setShowPassword] = useState(false);

  const {
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    mode: 'onTouched',
    defaultValues: { email: '', password: '' },
  });

  const onValid = (values: LoginFormValues) => {
    login.mutate(values, { onSuccess });
  };

  return (
    <Box component="form" onSubmit={handleSubmit(onValid)} noValidate>
      <Stack spacing={2.5}>
        <Controller
          name="email"
          control={control}
          render={({ field }) => (
            <TextField
              {...field}
              onChange={(e) => {
                field.onChange(e);
                if (login.isError) login.reset();
              }}
              label="Work email"
              type="email"
              fullWidth
              placeholder="you@company.com"
              error={Boolean(errors.email) || login.isError}
              helperText={errors.email?.message}
            />
          )}
        />

        <Stack spacing={0.75}>
          <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
            <Link component={RouterLink} to="/forgot-password" variant="body2" underline="hover">
              Forgot password?
            </Link>
          </Box>

          <Controller
            name="password"
            control={control}
            render={({ field }) => (
              <TextField
                {...field}
                onChange={(e) => {
                  field.onChange(e);
                  if (login.isError) login.reset();
                }}
                label="Password"
                type={showPassword ? 'text' : 'password'}
                fullWidth
                error={Boolean(errors.password) || login.isError}
                helperText={errors.password?.message}
                slotProps={{
                  input: {
                    endAdornment: (
                      <InputAdornment position="end">
                        <IconButton
                          aria-label={showPassword ? 'Hide password' : 'Show password'}
                          onClick={() => setShowPassword((v) => !v)}
                          edge="end"
                          size="small"
                        >
                          {showPassword ? <VisibilityOff fontSize="small" /> : <Visibility fontSize="small" />}
                        </IconButton>
                      </InputAdornment>
                    ),
                  },
                }}
              />
            )}
          />
        </Stack>

        {/* Unchecked by default: this product holds lease/payment data,
            so auto-opting into a persistent session is the wrong default
            for a financial product. UI-only for now either way — the
            backend always issues a fixed-length refresh token
            (JWT_REFRESH_TTL) regardless of this box; there's no
            variable-session-duration support server-side yet. */}
        <FormControlLabel
          control={<Checkbox size="small" />}
          label={<Typography variant="body2">Keep me signed in</Typography>}
        />

        {login.isError && <Alert severity="error">Invalid email or password.</Alert>}

        <Button type="submit" variant="contained" size="large" fullWidth disabled={login.isPending}>
          {login.isPending ? 'Signing in…' : 'Sign in'}
        </Button>
      </Stack>
    </Box>
  );
}
