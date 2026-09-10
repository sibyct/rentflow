import { useMutation } from '@tanstack/react-query';
import { authApi } from '../api/authApi';
import { useAuthStore } from '../store/authStore';
import type { LoginRequest, RegisterRequest } from '../types';

export function useLogin() {
  const setAuth = useAuthStore((s) => s.setAuth);
  return useMutation({
    mutationFn: (payload: LoginRequest) => authApi.login(payload),
    onSuccess: (result) => setAuth(result.accessToken, result.user),
  });
}

export function useRegister() {
  return useMutation({
    mutationFn: (payload: RegisterRequest) => authApi.register(payload),
  });
}

export function useLogout() {
  const clear = useAuthStore((s) => s.clear);
  return useMutation({
    mutationFn: () => authApi.logout(),
    onSettled: () => clear(),
  });
}
