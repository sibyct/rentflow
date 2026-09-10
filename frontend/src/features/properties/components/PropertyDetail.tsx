import { useNavigate } from 'react-router-dom';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Typography from '@mui/material/Typography';
import { QueryState } from '@/shared/components';
import { useDeleteProperty, useProperty } from '../hooks/usePropertyQueries';

interface PropertyDetailProps {
  propertyId: string;
}

export function PropertyDetail({ propertyId }: PropertyDetailProps) {
  const navigate = useNavigate();
  const { data: property, isLoading, isError, error, refetch } = useProperty(propertyId);
  const deleteProperty = useDeleteProperty();

  const handleDelete = () => {
    if (!window.confirm('Delete this property? This cannot be undone.')) return;
    deleteProperty.mutate(propertyId, { onSuccess: () => navigate('/properties') });
  };

  return (
    <QueryState isLoading={isLoading} isError={isError} error={error} onRetry={() => void refetch()}>
      {property && (
        <Box sx={{ maxWidth: 420 }}>
          <Typography variant="h6" sx={{ fontWeight: 600 }}>
            {property.address}
          </Typography>

          <Box
            component="dl"
            sx={{
              m: 0,
              mt: 2,
              display: 'grid',
              gridTemplateColumns: 'auto 1fr',
              columnGap: 2,
              rowGap: 1,
            }}
          >
            <Typography component="dt" variant="body2" color="text.secondary">
              Units
            </Typography>
            <Typography component="dd" variant="body2" sx={{ m: 0 }}>
              {property.unitCount}
            </Typography>

            <Typography component="dt" variant="body2" color="text.secondary">
              Status
            </Typography>
            <Typography component="dd" variant="body2" sx={{ m: 0 }}>
              {property.status}
            </Typography>

            <Typography component="dt" variant="body2" color="text.secondary">
              Created
            </Typography>
            <Typography component="dd" variant="body2" sx={{ m: 0 }}>
              {new Date(property.createdAt).toLocaleDateString()}
            </Typography>
          </Box>

          <Button
            variant="outlined"
            color="error"
            size="small"
            onClick={handleDelete}
            disabled={deleteProperty.isPending}
            sx={{ mt: 3 }}
          >
            {deleteProperty.isPending ? 'Deleting…' : 'Delete property'}
          </Button>
        </Box>
      )}
    </QueryState>
  );
}
