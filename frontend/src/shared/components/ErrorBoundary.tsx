import { Component, type ErrorInfo, type ReactNode } from 'react';
import Alert from '@mui/material/Alert';
import AlertTitle from '@mui/material/AlertTitle';
import Button from '@mui/material/Button';
import Box from '@mui/material/Box';

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: (error: Error, reset: () => void) => ReactNode;
}

interface ErrorBoundaryState {
  error: Error | null;
}

// React error boundaries must be class components — there is no hook
// equivalent for catching render errors in a subtree.
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  override state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  override componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Unhandled error in component tree:', error, info.componentStack);
  }

  reset = () => this.setState({ error: null });

  override render() {
    const { error } = this.state;
    if (!error) return this.props.children;

    if (this.props.fallback) return this.props.fallback(error, this.reset);

    return (
      <Box sx={{ m: 2 }}>
        <Alert severity="error">
          <AlertTitle>Something went wrong.</AlertTitle>
          {error.message}
          <Box sx={{ mt: 1 }}>
            <Button color="inherit" size="small" onClick={this.reset}>
              Try again
            </Button>
          </Box>
        </Alert>
      </Box>
    );
  }
}
