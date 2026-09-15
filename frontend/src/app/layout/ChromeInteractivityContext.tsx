import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';

interface ChromeInteractivityValue {
  dimmed: boolean;
  setDimmed: (dimmed: boolean) => void;
}

const ChromeInteractivityContext = createContext<ChromeInteractivityValue | null>(null);

/** Wraps AppShell so TopBar/Sidebar can read whether session-dependent chrome should be dimmed. */
export function ChromeInteractivityProvider({ children }: { children: ReactNode }) {
  const [dimmed, setDimmed] = useState(false);
  return (
    <ChromeInteractivityContext.Provider value={{ dimmed, setDimmed }}>
      {children}
    </ChromeInteractivityContext.Provider>
  );
}

function useChromeInteractivityContext(): ChromeInteractivityValue {
  const ctx = useContext(ChromeInteractivityContext);
  if (!ctx) throw new Error('useChromeInteractivityContext must be used within ChromeInteractivityProvider (see AppShell)');
  return ctx;
}

/** Read by TopBar/Sidebar to fade and disable the controls that need a live session. */
export function useChromeDimmed(): boolean {
  return useChromeInteractivityContext().dimmed;
}

/**
 * Dims the shell's session-dependent chrome (search, notifications, account
 * menu, nav links) for as long as the calling component stays mounted,
 * restoring it on unmount. ErrorState calls this via its `dimChrome` prop —
 * pages don't need to wire anything themselves.
 *
 * Reads the context directly (not via useChromeInteractivityContext,
 * which throws when absent): ErrorState — and anything built on it,
 * like ErrorBoundary's default fallback — can render above AppShell
 * (e.g. a top-level error boundary catching a crash before the shell
 * even mounts), where there's genuinely no chrome to dim. `dim=false`
 * is always safe; `dim=true` outside the provider is a silent no-op
 * rather than a hard crash.
 */
export function useDimChromeWhileMounted(dim: boolean): void {
  const ctx = useContext(ChromeInteractivityContext);
  useEffect(() => {
    if (!dim || !ctx) return;
    ctx.setDimmed(true);
    return () => ctx.setDimmed(false);
  }, [dim, ctx]);
}
