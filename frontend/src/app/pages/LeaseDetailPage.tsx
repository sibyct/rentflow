import { useParams } from 'react-router-dom';
import { LeaseDetailScreen } from '@/features/leases';

export function LeaseDetailPage() {
  const { id } = useParams<{ id: string }>();

  if (!id) return null;

  return <LeaseDetailScreen leaseId={id} />;
}
