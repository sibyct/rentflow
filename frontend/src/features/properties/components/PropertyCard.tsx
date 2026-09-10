import { Link as RouterLink } from 'react-router-dom';
import Card from '@mui/material/Card';
import CardActionArea from '@mui/material/CardActionArea';
import CardContent from '@mui/material/CardContent';
import Chip from '@mui/material/Chip';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import type { ChipProps } from '@mui/material/Chip';
import type { Property } from '../types';

const statusColor: Record<Property['status'], ChipProps['color']> = {
  active: 'success',
  inactive: 'default',
  maintenance: 'warning',
};

interface PropertyCardProps {
  property: Property;
}

export function PropertyCard({ property }: PropertyCardProps) {
  return (
    <Card variant="outlined">
      <CardActionArea component={RouterLink} to={`/properties/${property.id}`}>
        <CardContent>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'flex-start', justifyContent: 'space-between' }}>
            <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
              {property.address}
            </Typography>
            <Chip label={property.status} color={statusColor[property.status]} size="small" />
          </Stack>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
            {property.unitCount} {property.unitCount === 1 ? 'unit' : 'units'}
          </Typography>
        </CardContent>
      </CardActionArea>
    </Card>
  );
}
