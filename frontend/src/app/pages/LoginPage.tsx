import { Link as RouterLink, useNavigate } from 'react-router-dom';
import Box from '@mui/material/Box';
import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import { BrandMark } from '@/shared/components';
import { LoginForm } from '@/features/auth';

export function LoginPage() {
  const navigate = useNavigate();

  return (
    <Box>
      {/* AuthLayout already shows the brand mark in the left image panel
          on desktop (md+); this is only needed once that panel is
          hidden on mobile. */}
      <BrandMark sx={{ mb: 5, display: { xs: 'flex', md: 'none' } }} />

      <Typography variant="h3" sx={{ fontWeight: 700, fontSize: '1.75rem' }}>
        Sign in
      </Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mt: 1, mb: 4 }}>
        Manage your properties, leases and payments.
      </Typography>

      <LoginForm onSuccess={() => navigate('/properties')} />

      <Typography variant="body2" color="text.secondary" sx={{ mt: 4, textAlign: 'center' }}>
        Don&apos;t have an account? <Link component={RouterLink} to="/register">Create one</Link>
      </Typography>
    </Box>
  );
}
