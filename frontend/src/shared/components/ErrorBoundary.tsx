import { Component, Fragment, type ErrorInfo, type ReactNode } from 'react';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import ErrorOutlineOutlined from '@mui/icons-material/ErrorOutlineOutlined';
import { tokens } from '@/app/tokens';
import { ErrorState, type ErrorStateAction } from './ErrorState';

interface ErrorBoundaryProps {
  children: ReactNode;
  /** "Go to Dashboard", "Reload page", etc. — the fallback for when "Try again" doesn't fix it. Omit for none. */
  secondaryAction?: ErrorStateAction;
  /**
   * Fades the shell's session-dependent chrome while the fallback is
   * showing (see ErrorState). Appropriate for a boundary whose failure
   * really does mean the whole app is unusable (the top-level one in
   * App.tsx); leave off for anything scoped to one page or widget,
   * where the rest of the UI — nav included — should stay fully live.
   */
  dimChrome?: boolean;
  /**
   * Fully custom fallback in place of the default card — e.g. a
   * compact inline message for a small widget where a full ErrorState
   * card would be disproportionate. Called with the caught error and a
   * reset function that clears it and remounts the subtree.
   */
  fallback?: (error: Error, reset: () => void) => ReactNode;
}

interface ErrorBoundaryState {
  error: Error | null;
}

/**
 * Logs every caught error, in every environment — this is the one
 * place to wire a monitoring tool's client (Sentry, Bugsnag, ...) when
 * one is added:
 *
 *   Sentry.captureException(error, { contexts: { react: { componentStack: info.componentStack } } });
 *
 * console.error always fires alongside it (not instead of it) so
 * nothing is silently swallowed while that integration is pending.
 */
function reportError(error: Error, info: ErrorInfo): void {
  console.error('Unhandled error in component tree:', error, info.componentStack);
}

// React error boundaries must be class components — there is no hook
// equivalent for catching render errors in a subtree.
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  override state: ErrorBoundaryState = { error: null };

  // Remounts the subtree on reset, rather than just clearing the
  // boundary's own error flag: re-rendering the exact same
  // already-broken child instance in place usually throws again
  // immediately (its state, refs, and effects never reset). Bumping
  // this and keying the children with it forces a genuine fresh start.
  private resetKey = 0;

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  override componentDidCatch(error: Error, info: ErrorInfo) {
    reportError(error, info);
  }

  reset = () => {
    this.resetKey += 1;
    this.setState({ error: null });
  };

  override render() {
    const { error } = this.state;

    if (!error) {
      return <Fragment key={this.resetKey}>{this.props.children}</Fragment>;
    }

    if (this.props.fallback) return this.props.fallback(error, this.reset);

    const isDev = import.meta.env.DEV;

    return (
      <>
        <ErrorState
          icon={ErrorOutlineOutlined}
          title="Something went wrong"
          message={
            isDev
              ? error.message
              : "We hit a snag loading this. Our team's been notified — try again, or come back in a bit."
          }
          tone="neutral"
          dimChrome={this.props.dimChrome ?? false}
          primaryAction={{ label: 'Try again', onClick: this.reset }}
          secondaryAction={this.props.secondaryAction}
        />
        {isDev && error.stack && (
          <Box sx={{ maxWidth: 640, mx: 'auto', mt: -2, mb: 4, px: 3 }}>
            <Typography
              component="pre"
              sx={{
                fontFamily: tokens.fontMono,
                fontSize: 11.5,
                lineHeight: 1.6,
                color: tokens.slate[600],
                bgcolor: tokens.slate[50],
                border: `1px solid ${tokens.slate[200]}`,
                borderRadius: `${tokens.radiusControl}px`,
                p: 2,
                overflowX: 'auto',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
              }}
            >
              {error.stack}
            </Typography>
          </Box>
        )}
      </>
    );
  }
}
