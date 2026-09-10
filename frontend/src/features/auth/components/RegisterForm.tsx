import { useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { Controller, useForm } from 'react-hook-form';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import InputAdornment from '@mui/material/InputAdornment';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Visibility from '@mui/icons-material/Visibility';
import VisibilityOff from '@mui/icons-material/VisibilityOff';
import { useRegister } from '../hooks/useAuthMutations';
import { registerSchema, type RegisterFormValues } from '../schemas/authSchema';

interface RegisterFormProps {
  onSuccess?: () => void;
}

export function RegisterForm({ onSuccess }: RegisterFormProps) {
  const registerMutation = useRegister();
  const [showPassword, setShowPassword] = useState(false);

  const {
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<RegisterFormValues>({
    resolver: zodResolver(registerSchema),
    mode: 'onTouched',
    defaultValues: { email: '', password: '' },
  });

  const onValid = (values: RegisterFormValues) => {
    registerMutation.mutate(values, { onSuccess });
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
              label="Work email"
              type="email"
              fullWidth
              placeholder="you@company.com"
              error={Boolean(errors.email)}
              helperText={errors.email?.message}
            />
          )}
        />

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

        {registerMutation.isError && <Alert severity="error">Could not create the account.</Alert>}

        <Button type="submit" variant="contained" size="large" fullWidth disabled={registerMutation.isPending}>
          {registerMutation.isPending ? 'Creating account…' : 'Create account'}
        </Button>
      </Stack>
    </Box>
  );
}
