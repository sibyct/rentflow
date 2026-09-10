import { Link as RouterLink } from 'react-router-dom';
import Box from '@mui/material/Box';
import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import { BrandMark } from '@/shared/components';

/**
 * Deliberately not a working reset-request form: there is no
 * password-reset endpoint (email sending, reset tokens) on the backend
 * yet. Rendering a form that silently goes nowhere would be a dark
 * pattern, so this states the real situation instead.
 */
export function ForgotPasswordPage() {
  return (
    <Box>
      <BrandMark sx={{ mb: 5, display: { xs: 'flex', md: 'none' } }} />

      <Typography variant="h3" sx={{ fontWeight: 700, fontSize: '1.75rem' }}>
        Forgot your password?
      </Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mt: 1.5 }}>
        Self-service password reset isn&apos;t available yet. Contact your property manager or account
        administrator and they can reset it for you.
      </Typography>

      <Typography variant="body2" sx={{ mt: 4 }}>
        <Link component={RouterLink} to="/login">
          Back to sign in
        </Link>
      </Typography>
    </Box>
  );
}
