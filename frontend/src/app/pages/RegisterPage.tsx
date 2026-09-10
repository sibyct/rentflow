import { Link as RouterLink, useNavigate } from 'react-router-dom';
import Box from '@mui/material/Box';
import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import { BrandMark } from '@/shared/components';
import { RegisterForm } from '@/features/auth';

export function RegisterPage() {
  const navigate = useNavigate();

  return (
    <Box>
      <BrandMark sx={{ mb: 5, display: { xs: 'flex', md: 'none' } }} />

      <Typography variant="h3" sx={{ fontWeight: 700, fontSize: '1.75rem' }}>
        Create your account
      </Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mt: 1, mb: 4 }}>
        Start managing your properties in minutes.
      </Typography>

      <RegisterForm onSuccess={() => navigate('/login')} />

      <Typography variant="body2" color="text.secondary" sx={{ mt: 4, textAlign: 'center' }}>
        Already have an account? <Link component={RouterLink} to="/login">Sign in</Link>
      </Typography>
    </Box>
  );
}
