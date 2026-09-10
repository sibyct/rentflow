import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';

interface LoadingSpinnerProps {
  label?: string;
}

export function LoadingSpinner({ label = 'Loading…' }: LoadingSpinnerProps) {
  return (
    <Stack
      direction="row"
      spacing={1.5}
      role="status"
      sx={{ alignItems: 'center', justifyContent: 'center', py: 4 }}
    >
      <CircularProgress size={20} />
      <Typography variant="body2" color="text.secondary">
        {label}
      </Typography>
    </Stack>
  );
}
