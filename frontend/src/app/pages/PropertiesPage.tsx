import { Link as RouterLink } from 'react-router-dom';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { PropertyList } from '@/features/properties';

export function PropertiesPage() {
  return (
    <Box>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 3 }}>
        <Typography variant="h5" component="h1" sx={{ fontWeight: 600 }}>
          Properties
        </Typography>
        <Button component={RouterLink} to="/properties/new" variant="contained" disableElevation>
          Add property
        </Button>
      </Stack>
      <PropertyList />
    </Box>
  );
}
