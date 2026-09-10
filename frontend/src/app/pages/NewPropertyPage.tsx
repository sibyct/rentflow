import { useNavigate } from 'react-router-dom';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { PropertyForm, useCreateProperty } from '@/features/properties';

export function NewPropertyPage() {
  const navigate = useNavigate();
  const createProperty = useCreateProperty();

  return (
    <Box>
      <Typography variant="h5" component="h1" sx={{ mb: 3, fontWeight: 600 }}>
        Add property
      </Typography>
      <PropertyForm
        isSubmitting={createProperty.isPending}
        submitError={createProperty.error}
        onSubmit={(input) =>
          createProperty.mutate(input, {
            onSuccess: (property) => navigate(`/properties/${property.id}`),
          })
        }
      />
    </Box>
  );
}
