import { useParams } from 'react-router-dom';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { PropertyDetail } from '@/features/properties';

export function PropertyDetailPage() {
  const { id } = useParams<{ id: string }>();

  if (!id) return null;

  return (
    <Box>
      <Typography variant="h5" component="h1" sx={{ mb: 3, fontWeight: 600 }}>
        Property details
      </Typography>
      <PropertyDetail propertyId={id} />
    </Box>
  );
}
