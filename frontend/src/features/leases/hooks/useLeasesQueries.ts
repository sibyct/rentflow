import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query';
import { unitsQueryKeys } from '@/features/units/hooks/useUnitsQueries';
import { leasesApi, type LeasePortfolioListParams } from '../api/leasesApi';
import type { LeaseFormValues } from '../schemas/leaseSchema';

export const leasesQueryKeys = {
  all: ['leases'] as const,
  detail: (id: string) => [...leasesQueryKeys.all, 'detail', id] as const,
  portfolio: (params: LeasePortfolioListParams) => [...leasesQueryKeys.all, 'portfolio', params] as const,
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
    mutationFn: ({ id, values }: { id: string; values: LeaseFormValues }) => leasesApi.update(id, values),
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
