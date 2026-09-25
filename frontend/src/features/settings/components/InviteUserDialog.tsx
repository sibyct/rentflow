import { useEffect, useState } from 'react';
import { Controller, useForm, useWatch } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import FormControlLabel from '@mui/material/FormControlLabel';
import IconButton from '@mui/material/IconButton';
import MenuItem from '@mui/material/MenuItem';
import Radio from '@mui/material/Radio';
import RadioGroup from '@mui/material/RadioGroup';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import CloseOutlined from '@mui/icons-material/CloseOutlined';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import { useProperties } from '@/features/properties/hooks/usePropertiesQueries';
import { StatusChip } from '@/shared/components';
import { useInviteStaff, useResendStaffInvite, useReactivateStaff } from '../hooks/useStaffQueries';
import { inviteDefaultValues, inviteSchema, type InviteFormValues } from '../schemas/inviteSchema';
import { initials } from '../utils';
import { ROLE_LABELS, STAFF_ROLES, USER_STATUS_LABELS, USER_STATUS_TONE, type StaffUser } from '../types';

interface Toast {
  message: string;
  severity: 'success' | 'error';
}

interface InviteUserDialogProps {
  open: boolean;
  onClose: () => void;
  onToast: (toast: Toast) => void;
}

function errorMessage(err: unknown): string {
  return err instanceof ApiError ? err.message : 'Something went wrong. Please try again.';
}

/**
 * Duplicate-email handling: rather than letting a second invite silently
 * create a second record, a matched email replaces the form with a small
 * conflict panel naming the existing user and offering the one action
 * that actually applies to their current status.
 */
export function InviteUserDialog({ open, onClose, onToast }: InviteUserDialogProps) {
  const invite = useInviteStaff();
  const resendInvite = useResendStaffInvite();
  const reactivate = useReactivateStaff();
  // A generous limit, same as the backend's own ownedPropertyNames lookup
  // (StaffService) — this picker needs every property, not a page of them.
  const { data: propertiesResult } = useProperties({ limit: 500, offset: 0 });
  const properties = propertiesResult?.properties ?? [];

  const [conflict, setConflict] = useState<StaffUser | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const {
    control,
    handleSubmit,
    reset,
    setValue,
    formState: { errors },
  } = useForm<InviteFormValues>({ resolver: zodResolver(inviteSchema), defaultValues: inviteDefaultValues() });

  const role = useWatch({ control, name: 'role' });
  const accessAll = useWatch({ control, name: 'accessAll' });
  const selectedPropertyIds = useWatch({ control, name: 'properties' });

  useEffect(() => {
    if (open) {
      reset(inviteDefaultValues());
      setConflict(null);
      setSubmitError(null);
    }
  }, [open, reset]);

  function toggleProperty(id: string) {
    const next = selectedPropertyIds.includes(id) ? selectedPropertyIds.filter((p) => p !== id) : [...selectedPropertyIds, id];
    setValue('properties', next, { shouldValidate: true });
  }

  function onSubmit(values: InviteFormValues) {
    setSubmitError(null);
    invite.mutate(
      {
        name: values.name,
        email: values.email,
        role: values.role,
        propertyAccess: { all: values.accessAll, propertyIds: values.properties },
      },
      {
        onSuccess: (result) => {
          if (result.kind === 'conflict') {
            setConflict(result.existing);
            return;
          }
          onToast({ message: `Invitation sent to ${result.user.email}`, severity: 'success' });
          onClose();
        },
        onError: (err) => setSubmitError(errorMessage(err)),
      },
    );
  }

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', pr: 1 }}>
        <DialogTitle sx={{ pb: 1 }}>Invite user</DialogTitle>
        <IconButton onClick={onClose} aria-label="Close" sx={{ mr: 1 }}>
          <CloseOutlined fontSize="small" />
        </IconButton>
      </Stack>

      {conflict ? (
        <>
          <DialogContent>
            <Alert severity="warning" sx={{ mb: 2.5 }}>
              Someone with this email already has an account — inviting them again would create a duplicate.
            </Alert>
            <Stack direction="row" sx={{ alignItems: 'center', gap: 1.5, p: 2, border: `1px solid ${tokens.slate[200]}`, borderRadius: `${tokens.radiusControl}px` }}>
              <Box sx={{ width: 34, height: 34, borderRadius: '50%', bgcolor: tokens.slate[700], color: 'common.white', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 12, fontWeight: 700, flexShrink: 0 }}>
                {initials(conflict.name)}
              </Box>
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <Typography sx={{ fontSize: 13.5, fontWeight: 600 }}>{conflict.name}</Typography>
                <Typography sx={{ fontSize: 12, color: tokens.slate[500] }}>{conflict.email}</Typography>
              </Box>
              <StatusChip label={USER_STATUS_LABELS[conflict.status]} tone={USER_STATUS_TONE[conflict.status]} />
            </Stack>
            {conflict.status === 'active' && (
              <Typography sx={{ fontSize: 13, color: tokens.slate[600], mt: 2 }}>
                This person already has an active account — there&apos;s nothing to resend.
              </Typography>
            )}
          </DialogContent>
          <DialogActions sx={{ p: 2.5, pt: 0 }}>
            <Button variant="text" onClick={onClose} sx={{ color: tokens.slate[600] }}>
              Close
            </Button>
            {(conflict.status === 'invited' || conflict.status === 'invite_expired') && (
              <Button
                variant="contained"
                disabled={resendInvite.isPending}
                onClick={() =>
                  resendInvite.mutate(conflict.id, {
                    onSuccess: () => {
                      onToast({ message: `Invitation resent to ${conflict.email}`, severity: 'success' });
                      onClose();
                    },
                    onError: (err) => onToast({ message: errorMessage(err), severity: 'error' }),
                  })
                }
              >
                Resend invite
              </Button>
            )}
            {conflict.status === 'deactivated' && (
              <Button
                variant="contained"
                disabled={reactivate.isPending}
                onClick={() =>
                  reactivate.mutate(conflict.id, {
                    onSuccess: () => {
                      onToast({ message: `${conflict.name} was reactivated`, severity: 'success' });
                      onClose();
                    },
                    onError: (err) => onToast({ message: errorMessage(err), severity: 'error' }),
                  })
                }
              >
                Reactivate
              </Button>
            )}
          </DialogActions>
        </>
      ) : (
        <form onSubmit={handleSubmit(onSubmit)} noValidate>
          <DialogContent>
            <Stack spacing={2.5}>
              {submitError && (
                <Alert severity="error" onClose={() => setSubmitError(null)}>
                  {submitError}
                </Alert>
              )}
              <Controller
                name="name"
                control={control}
                render={({ field }) => (
                  <TextField {...field} label="Name" required autoFocus error={Boolean(errors.name)} helperText={errors.name?.message} />
                )}
              />
              <Controller
                name="email"
                control={control}
                render={({ field }) => (
                  <TextField {...field} label="Email" type="email" required error={Boolean(errors.email)} helperText={errors.email?.message} />
                )}
              />
              <Controller
                name="role"
                control={control}
                render={({ field }) => (
                  <TextField {...field} value={field.value ?? ''} select label="Role" required error={Boolean(errors.role)} helperText={errors.role?.message}>
                    {STAFF_ROLES.map((r) => (
                      <MenuItem key={r} value={r}>
                        {ROLE_LABELS[r]}
                      </MenuItem>
                    ))}
                  </TextField>
                )}
              />

              {role === 'admin' ? (
                <Alert severity="info" sx={{ fontSize: 13 }}>
                  Admins always have access to every property.
                </Alert>
              ) : (
                <Box>
                  <Typography sx={{ fontSize: 12.5, fontWeight: 600, color: tokens.slate[600], mb: 1 }}>Property access</Typography>
                  <RadioGroup
                    value={accessAll ? 'all' : 'specific'}
                    onChange={(e) => setValue('accessAll', e.target.value === 'all', { shouldValidate: true })}
                  >
                    <FormControlLabel value="all" control={<Radio size="small" />} label={<Typography sx={{ fontSize: 13.5 }}>All properties</Typography>} />
                    <FormControlLabel value="specific" control={<Radio size="small" />} label={<Typography sx={{ fontSize: 13.5 }}>Specific properties</Typography>} />
                  </RadioGroup>
                  {!accessAll && (
                    <Box sx={{ mt: 1, ml: 4, display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)' }}>
                      {properties.map((p) => (
                        <FormControlLabel
                          key={p.id}
                          control={<Checkbox size="small" checked={selectedPropertyIds.includes(p.id)} onChange={() => toggleProperty(p.id)} />}
                          label={<Typography sx={{ fontSize: 13 }}>{p.name}</Typography>}
                        />
                      ))}
                    </Box>
                  )}
                  {errors.properties && (
                    <Typography sx={{ fontSize: 12, color: 'error.main', mt: 0.5, ml: 4 }}>{errors.properties.message}</Typography>
                  )}
                </Box>
              )}
            </Stack>
          </DialogContent>
          <DialogActions sx={{ p: 2.5, pt: 0 }}>
            <Button variant="text" onClick={onClose} sx={{ color: tokens.slate[600] }}>
              Cancel
            </Button>
            <Button type="submit" variant="contained" disabled={invite.isPending} startIcon={invite.isPending ? <CircularProgress size={15} color="inherit" /> : undefined}>
              {invite.isPending ? 'Sending…' : 'Send invite'}
            </Button>
          </DialogActions>
        </form>
      )}
    </Dialog>
  );
}
