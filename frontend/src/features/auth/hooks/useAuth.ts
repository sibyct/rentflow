import { useAuthStore } from '../store/authStore';

/** Read-only view of the current session, for components that just need to know who's logged in. */
export function useAuth() {
  const user = useAuthStore((s) => s.user);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return { user, isAuthenticated };
}
