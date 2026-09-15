import { useQuery } from '@tanstack/react-query';
import { authApi } from '../api/authApi';
import { useAuthStore } from '../store/authStore';

/**
 * Exchanges the httpOnly refresh cookie (if any) for a fresh access
 * token once, on app boot — see App.tsx's SessionBootstrap, which shows
 * AppLoadingScreen for as long as this is pending. Not finding a valid
 * cookie just means "start logged out," so that failure is swallowed
 * here rather than surfaced or retried.
 *
 * The setAuth side effect lives inside queryFn rather than an onSuccess
 * callback: React Query v5 removed onSuccess/onError from useQuery, and
 * the call site only ever needs isPending (never the token itself), so
 * there's nothing for a separate useEffect-on-data to do that queryFn
 * doesn't already cover in one pass.
 */
export function useSessionBootstrap() {
  const setAuth = useAuthStore((s) => s.setAuth);

  return useQuery({
    queryKey: ['auth', 'bootstrap'],
    queryFn: async () => {
      try {
        const result = await authApi.refresh();
        setAuth(result.accessToken, result.user);
      } catch {
        // No valid session cookie yet — starting logged out is expected.
      }
      return null;
    },
    retry: false,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
}
