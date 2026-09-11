import { useParams } from 'react-router-dom';
import { PropertyDetail } from '@/features/properties';

export function PropertyDetailPage() {
  const { id } = useParams<{ id: string }>();

  if (!id) return null;

  return <PropertyDetail propertyId={id} />;
}
