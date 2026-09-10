import type { ReactNode } from 'react';
import { LoadingSpinner } from './LoadingSpinner';
import { ErrorMessage } from './ErrorMessage';

interface QueryStateProps {
  isLoading: boolean;
  isError: boolean;
  error?: unknown;
  onRetry?: () => void;
  children: ReactNode;
}

/**
 * The loading/error UI pattern reused across every feature that reads
 * data with React Query: render a spinner while loading, an error
 * message with a retry action on failure, and the real content once
 * data is available. Keeps individual list/detail components free of
 * repeated isLoading/isError branching.
 */
export function QueryState({ isLoading, isError, error, onRetry, children }: QueryStateProps) {
  if (isLoading) return <LoadingSpinner />;
  if (isError) return <ErrorMessage error={error} onRetry={onRetry} />;
  return <>{children}</>;
}
