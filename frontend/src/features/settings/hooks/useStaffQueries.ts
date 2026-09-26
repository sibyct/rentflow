import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query';
import { staffApi, type InviteInput, type UpdateStaffInput } from '../api/staffApi';

export const staffQueryKeys = {
  all: ['staff'] as const,
  list: () => [...staffQueryKeys.all, 'list'] as const,
  audit: (limit: number, offset: number) => [...staffQueryKeys.all, 'audit', limit, offset] as const,
};

export function useStaff() {
  return useQuery({
    queryKey: staffQueryKeys.list(),
    queryFn: () => staffApi.list(),
  });
}

export function useStaffAuditLog(limit: number, offset: number) {
  return useQuery({
    queryKey: staffQueryKeys.audit(limit, offset),
    queryFn: () => staffApi.auditLog(limit, offset),
    placeholderData: (previousData) => previousData,
  });
}

/** Backs the (unauthenticated) accept-invite page — token comes from the email link's ?token= query param. */
export function useInviteLookup(token: string) {
  return useQuery({
    queryKey: ['invite-lookup', token],
    queryFn: () => staffApi.lookupInvite(token),
    enabled: Boolean(token),
    retry: false,
  });
}

/** Backs the (unauthenticated) reset-password page — same shape as useInviteLookup, for the admin-triggered reset-password link. */
export function usePasswordResetLookup(token: string) {
  return useQuery({
    queryKey: ['password-reset-lookup', token],
    queryFn: () => staffApi.lookupPasswordReset(token),
    enabled: Boolean(token),
    retry: false,
  });
}

function invalidateAfterStaffChange(queryClient: QueryClient) {
  void queryClient.invalidateQueries({ queryKey: staffQueryKeys.all });
}

export function useInviteStaff() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: InviteInput) => staffApi.invite(input),
    onSuccess: () => invalidateAfterStaffChange(queryClient),
  });
}

export function useUpdateStaff() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateStaffInput }) => staffApi.update(id, input),
    onSuccess: () => invalidateAfterStaffChange(queryClient),
  });
}

export function useDeactivateStaff() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => staffApi.deactivate(id),
    onSuccess: () => invalidateAfterStaffChange(queryClient),
  });
}

export function useReactivateStaff() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => staffApi.reactivate(id),
    onSuccess: () => invalidateAfterStaffChange(queryClient),
  });
}

export function useResendStaffInvite() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => staffApi.resendInvite(id),
    onSuccess: () => invalidateAfterStaffChange(queryClient),
  });
}

export function useResetStaffPassword() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => staffApi.resetPassword(id),
    onSuccess: () => invalidateAfterStaffChange(queryClient),
  });
}
