import { ErrorState, ERROR_STATE_PRESETS } from '@/shared/components';

export function NotFoundPage() {
  return <ErrorState {...ERROR_STATE_PRESETS.notFound} primaryAction={{ label: 'Go to dashboard', href: '/dashboard' }} />;
}
