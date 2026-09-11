import { Link as RouterLink } from 'react-router-dom';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import { PropertyList } from '@/features/properties';

export function PropertiesPage() {
  return (
    <Box>
      <Stack direction="row" sx={{ justifyContent: 'flex-end', mb: 2 }}>
        <Button component={RouterLink} to="/properties/new" variant="contained" disableElevation>
          Add property
        </Button>
      </Stack>
      <PropertyList />
    </Box>
  );
}
