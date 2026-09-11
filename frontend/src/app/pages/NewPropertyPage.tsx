import { useNavigate } from 'react-router-dom';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { PropertyForm, useCreateProperty } from '@/features/properties';

export function NewPropertyPage() {
  const navigate = useNavigate();
  const createProperty = useCreateProperty();

  return (
    <Box>
      <Typography variant="body1" color="text.secondary" sx={{ mb: 3 }}>
        Add a new property
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
