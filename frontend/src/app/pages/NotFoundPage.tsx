import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';

export function NotFoundPage() {
  return (
    <Box sx={{ py: 8, textAlign: 'center' }}>
      <Typography variant="h5" component="h1" sx={{ fontWeight: 600 }}>
        Page not found
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
        The page you&apos;re looking for doesn&apos;t exist.
      </Typography>
    </Box>
  );
}
