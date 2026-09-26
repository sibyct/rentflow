import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query';
import { unitsQueryKeys } from '@/features/units/hooks/useUnitsQueries';
import {
  leasesApi,
  type ChangeRentInput,
  type CorrectTerminationInput,
  type GenerateRenewalInput,
  type LeasePortfolioListParams,
  type TerminateLeaseInput,
} from '../api/leasesApi';
import type { LeaseFormValues } from '../schemas/leaseSchema';
import type { LeaseDocumentCategory, LeaseStatus } from '../types';

export const leasesQueryKeys = {
  all: ['leases'] as const,
  detail: (id: string) => [...leasesQueryKeys.all, 'detail', id] as const,
  portfolio: (params: LeasePortfolioListParams) => [...leasesQueryKeys.all, 'portfolio', params] as const,
  rentHistory: (id: string) => [...leasesQueryKeys.all, 'rentHistory', id] as const,
  audit: (id: string) => [...leasesQueryKeys.all, 'audit', id] as const,
  documents: (id: string) => [...leasesQueryKeys.all, 'documents', id] as const,
};

export function useLeasesPortfolio(params: LeasePortfolioListParams, options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: leasesQueryKeys.portfolio(params),
    queryFn: () => leasesApi.listForOwner(params),
    placeholderData: (previousData) => previousData,
    enabled: options.enabled ?? true,
  });
}

export function useLease(id: string | undefined) {
  return useQuery({
    queryKey: leasesQueryKeys.detail(id ?? ''),
    queryFn: () => leasesApi.get(id!),
    enabled: Boolean(id),
  });
}

// A lease change also invalidates the unit's own queries: a unit's
// occupancy/tenant picture is tied to its active lease (see
// UnitDetailScreen's lease cross-link), and the global Units page's
// stats read from the same underlying data.
function invalidateAfterLeaseChange(queryClient: QueryClient, unitId: string) {
  void queryClient.invalidateQueries({ queryKey: leasesQueryKeys.all });
  void queryClient.invalidateQueries({ queryKey: unitsQueryKeys.detail(unitId) });
}

export function useCreateLease(unitId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (values: LeaseFormValues) => leasesApi.create(unitId, values),
    onSuccess: () => invalidateAfterLeaseChange(queryClient, unitId),
  });
}

export function useUpdateLease(unitId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, values, originalStatus }: { id: string; values: LeaseFormValues; originalStatus: LeaseStatus }) =>
      leasesApi.update(id, values, originalStatus),
    onSuccess: () => invalidateAfterLeaseChange(queryClient, unitId),
  });
}

export function useDeleteLease(unitId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => leasesApi.delete(id),
    onSuccess: () => invalidateAfterLeaseChange(queryClient, unitId),
  });
}

export function useLeaseRentHistory(leaseId: string | undefined) {
  return useQuery({
    queryKey: leasesQueryKeys.rentHistory(leaseId ?? ''),
    queryFn: () => leasesApi.rentHistory(leaseId!),
    enabled: Boolean(leaseId),
  });
}

/** The only way to change an active lease's effective rent (R3/R4/R6) — always appends a dated amendment. */
export function useChangeRent(leaseId: string, unitId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: ChangeRentInput) => leasesApi.changeRent(leaseId, input),
    onSuccess: () => {
      invalidateAfterLeaseChange(queryClient, unitId);
      void queryClient.invalidateQueries({ queryKey: leasesQueryKeys.rentHistory(leaseId) });
      void queryClient.invalidateQueries({ queryKey: leasesQueryKeys.audit(leaseId) });
    },
  });
}

/** Turns an Accepted renewal into a new, real Lease record (R7) — invalidates the whole portfolio since a second lease now exists. */
export function useGenerateRenewalLease(leaseId: string, unitId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: GenerateRenewalInput) => leasesApi.generateRenewal(leaseId, input),
    onSuccess: () => {
      invalidateAfterLeaseChange(queryClient, unitId);
      void queryClient.invalidateQueries({ queryKey: leasesQueryKeys.audit(leaseId) });
    },
  });
}

/** The dedicated action for ending a lease (R10/R15) — always vacates the unit. */
export function useTerminateLease(leaseId: string, unitId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: TerminateLeaseInput) => leasesApi.terminate(leaseId, input),
    onSuccess: () => {
      invalidateAfterLeaseChange(queryClient, unitId);
      void queryClient.invalidateQueries({ queryKey: leasesQueryKeys.audit(leaseId) });
      void queryClient.invalidateQueries({ queryKey: unitsQueryKeys.detail(unitId) });
    },
  });
}

/** Fixes a mistake in an already-terminated lease's termination details — a separate, reason-required action, distinct from Terminate. Never reachable via a generic field edit. */
export function useCorrectTermination(leaseId: string, unitId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CorrectTerminationInput) => leasesApi.correctTermination(leaseId, input),
    onSuccess: () => {
      invalidateAfterLeaseChange(queryClient, unitId);
      void queryClient.invalidateQueries({ queryKey: leasesQueryKeys.audit(leaseId) });
    },
  });
}

export function useLeaseAudit(leaseId: string | undefined) {
  return useQuery({
    queryKey: leasesQueryKeys.audit(leaseId ?? ''),
    queryFn: () => leasesApi.audit(leaseId!),
    enabled: Boolean(leaseId),
  });
}

export function useLeaseDocuments(leaseId: string | undefined) {
  return useQuery({
    queryKey: leasesQueryKeys.documents(leaseId ?? ''),
    queryFn: () => leasesApi.documents(leaseId!),
    enabled: Boolean(leaseId),
  });
}

export function useAddLeaseDocument(leaseId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ attachmentId, category }: { attachmentId: string; category: LeaseDocumentCategory }) =>
      leasesApi.addDocument(leaseId, attachmentId, category),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: leasesQueryKeys.documents(leaseId) }),
  });
}

export function useDeleteLeaseDocument(leaseId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (documentId: string) => leasesApi.deleteDocument(leaseId, documentId),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: leasesQueryKeys.documents(leaseId) }),
  });
}
