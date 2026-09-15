import { useParams } from 'react-router-dom';
import { PropertyDetailScreen } from '@/features/properties/components/PropertyDetailScreen';

export function PropertyDetailPage() {
  const { id } = useParams<{ id: string }>();

  if (!id) return null;

  return <PropertyDetailScreen propertyId={id} />;
}
