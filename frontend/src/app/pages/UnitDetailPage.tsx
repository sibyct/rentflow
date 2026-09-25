import { useParams } from 'react-router-dom';
import { UnitDetailScreen } from '@/features/units/components/UnitDetailScreen';

export function UnitDetailPage() {
  const { id } = useParams<{ id: string }>();

  if (!id) return null;

  return <UnitDetailScreen unitId={id} />;
}
