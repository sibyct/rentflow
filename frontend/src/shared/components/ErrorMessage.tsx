import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import { ApiError } from '@/api/client';

interface ErrorMessageProps {
  error: unknown;
  onRetry?: () => void;
}

function messageFor(error: unknown): string {
  if (error instanceof ApiError) return error.message;
  if (error instanceof Error) return error.message;
  return 'Something went wrong. Please try again.';
}

export function ErrorMessage({ error, onRetry }: ErrorMessageProps) {
  return (
    <Alert
      severity="error"
      action={
        onRetry && (
          <Button color="inherit" size="small" onClick={onRetry}>
            Try again
          </Button>
        )
      }
    >
      {messageFor(error)}
    </Alert>
  );
}
