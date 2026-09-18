import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query';
import type { WorkOrderCategory } from '@/features/maintenance/types';
import { vendorsApi, type VendorListParams } from '../api/vendorsApi';
import type { VendorFormValues } from '../schemas/vendorSchema';

export const vendorsQueryKeys = {
  all: ['vendors'] as const,
  list: (params: VendorListParams) => [...vendorsQueryKeys.all, 'list', params] as const,
  forCategory: (category: WorkOrderCategory | '') => [...vendorsQueryKeys.all, 'category', category] as const,
  detail: (id: string) => [...vendorsQueryKeys.all, 'detail', id] as const,
  propertiesServed: (id: string) => [...vendorsQueryKeys.all, 'properties-served', id] as const,
  spendSummary: (id: string) => [...vendorsQueryKeys.all, 'spend', id] as const,
};

export function useVendors(params: VendorListParams) {
  return useQuery({
    queryKey: vendorsQueryKeys.list(params),
    queryFn: () => vendorsApi.list(params),
    placeholderData: (previousData) => previousData,
  });
}

/** Backs the Vendor Selector inside the work order drawer. */
export function useVendorsForCategory(category: WorkOrderCategory | undefined) {
  return useQuery({
    queryKey: vendorsQueryKeys.forCategory(category ?? ''),
    queryFn: () => vendorsApi.listForCategory(category!),
    enabled: Boolean(category),
  });
}

export function useVendor(id: string | undefined) {
  return useQuery({
    queryKey: vendorsQueryKeys.detail(id ?? ''),
    queryFn: () => vendorsApi.get(id!),
    enabled: Boolean(id),
  });
}

export function useVendorPropertiesServed(id: string | undefined, options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: vendorsQueryKeys.propertiesServed(id ?? ''),
    queryFn: () => vendorsApi.getPropertiesServed(id!),
    enabled: Boolean(id) && (options.enabled ?? true),
  });
}

export function useVendorSpendSummary(id: string | undefined) {
  return useQuery({
    queryKey: vendorsQueryKeys.spendSummary(id ?? ''),
    queryFn: () => vendorsApi.getSpendSummary(id!),
    enabled: Boolean(id),
  });
}

function invalidateAfterVendorChange(queryClient: QueryClient) {
  void queryClient.invalidateQueries({ queryKey: vendorsQueryKeys.all });
}

export function useCreateVendor() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (values: VendorFormValues) => vendorsApi.create(values),
    onSuccess: () => invalidateAfterVendorChange(queryClient),
  });
}

export function useUpdateVendor() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: VendorFormValues }) => vendorsApi.update(id, values),
    onSuccess: () => invalidateAfterVendorChange(queryClient),
  });
}

export function useDeleteVendor() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => vendorsApi.delete(id),
    onSuccess: () => invalidateAfterVendorChange(queryClient),
  });
}
