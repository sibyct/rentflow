import { Link as RouterLink, Outlet } from 'react-router-dom';
import AppBar from '@mui/material/AppBar';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Container from '@mui/material/Container';
import Stack from '@mui/material/Stack';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';
import { useAuth, useLogout } from '@/features/auth';
import { BrandMark } from '@/shared/components';

export function RootLayout() {
  const { isAuthenticated, user } = useAuth();
  const logout = useLogout();

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'grey.50' }}>
      <AppBar position="static" color="inherit" elevation={0} sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Container maxWidth="lg">
          <Toolbar disableGutters sx={{ justifyContent: 'space-between' }}>
            <Box component={RouterLink} to="/" sx={{ textDecoration: 'none' }}>
              <BrandMark size={28} />
            </Box>

            <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
              {isAuthenticated ? (
                <>
                  <Button component={RouterLink} to="/properties" color="inherit">
                    Properties
                  </Button>
                  <Typography variant="body2" color="text.secondary" sx={{ px: 1 }}>
                    {user?.email}
                  </Typography>
                  <Button color="inherit" onClick={() => logout.mutate()}>
                    Sign out
                  </Button>
                </>
              ) : (
                <>
                  <Button component={RouterLink} to="/login" color="inherit">
                    Sign in
                  </Button>
                  <Button component={RouterLink} to="/register" variant="contained" disableElevation>
                    Create account
                  </Button>
                </>
              )}
            </Stack>
          </Toolbar>
        </Container>
      </AppBar>

      <Container maxWidth="lg" sx={{ py: 4 }}>
        <Outlet />
      </Container>
    </Box>
  );
}
