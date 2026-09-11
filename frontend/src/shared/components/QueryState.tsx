import type { ReactNode } from 'react';
import { LoadingSpinner } from './LoadingSpinner';
import { ErrorMessage } from './ErrorMessage';
import { ErrorState } from './ErrorState';
import { ERROR_STATE_PRESETS } from './errorStatePresets';
import { ApiError } from '@/api/client';

interface QueryStateProps {
  isLoading: boolean;
  isError: boolean;
  error?: unknown;
  onRetry?: () => void;
  children: ReactNode;
}

/**
 * The loading/error UI pattern reused across every feature that reads
 * data with React Query: render a spinner while loading, the real content
 * once data is available, and — on failure — the error state that
 * actually matches what happened, not a one-size-fits-all alert.
 * `apiClient` throws a 401 as an ApiError once its own refresh attempt
 * has already failed (see api/client.ts), so a 401 here always means
 * "log in again," never "retry."
 */
export function QueryState({ isLoading, isError, error, onRetry, children }: QueryStateProps) {
  if (isLoading) return <LoadingSpinner />;
  if (isError) return <QueryError error={error} onRetry={onRetry} />;
  return <>{children}</>;
}

function QueryError({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  if (error instanceof ApiError) {
    if (error.status === 401) {
      return <ErrorState {...ERROR_STATE_PRESETS.sessionExpired} primaryAction={{ label: 'Log in', href: '/login' }} />;
    }
    if (error.status === 403) {
      return <ErrorState {...ERROR_STATE_PRESETS.forbidden} primaryAction={{ label: 'Go to dashboard', href: '/dashboard' }} />;
    }
    if (error.status >= 500) {
      return (
        <ErrorState
          {...ERROR_STATE_PRESETS.serverError}
          primaryAction={onRetry ? { label: 'Try again', onClick: onRetry } : undefined}
          secondaryAction={{ label: 'Contact support' }}
        />
      );
    }
  }

  if (typeof navigator !== 'undefined' && !navigator.onLine) {
    return <ErrorState {...ERROR_STATE_PRESETS.offline} primaryAction={{ label: 'Reload page', onClick: () => window.location.reload() }} />;
  }

  return <ErrorMessage error={error} onRetry={onRetry} />;
}
