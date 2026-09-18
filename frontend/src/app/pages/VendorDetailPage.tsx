import { useParams } from 'react-router-dom';
import { VendorDetailScreen } from '@/features/vendors';

export function VendorDetailPage() {
  const { id } = useParams<{ id: string }>();

  if (!id) return null;

  return <VendorDetailScreen vendorId={id} />;
}
