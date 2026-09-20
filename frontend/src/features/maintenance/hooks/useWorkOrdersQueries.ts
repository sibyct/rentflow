import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query';
import type { MaintenanceVisibility, WorkOrderStatus } from '../types';
import { workOrdersApi, type WorkOrderListParams } from '../api/workOrdersApi';
import type { WorkOrderFormValues } from '../schemas/workOrderSchema';

export const workOrdersQueryKeys = {
  all: ['workOrders'] as const,
  list: (params: WorkOrderListParams) => [...workOrdersQueryKeys.all, 'list', params] as const,
  detail: (id: string) => [...workOrdersQueryKeys.all, 'detail', id] as const,
  summary: () => [...workOrdersQueryKeys.all, 'summary'] as const,
  activity: (id: string) => [...workOrdersQueryKeys.all, 'activity', id] as const,
  recentActivity: (limit: number, offset: number) => [...workOrdersQueryKeys.all, 'recent-activity', limit, offset] as const,
};

export function useWorkOrders(params: WorkOrderListParams) {
  return useQuery({
    queryKey: workOrdersQueryKeys.list(params),
    queryFn: () => workOrdersApi.list(params),
    placeholderData: (previousData) => previousData,
  });
}

export function useWorkOrder(id: string | undefined) {
  return useQuery({
    queryKey: workOrdersQueryKeys.detail(id ?? ''),
    queryFn: () => workOrdersApi.get(id!),
    enabled: Boolean(id),
  });
}

export function useWorkOrderSummary() {
  return useQuery({
    queryKey: workOrdersQueryKeys.summary(),
    queryFn: () => workOrdersApi.getSummary(),
  });
}

export function useWorkOrderActivity(workOrderId: string | undefined) {
  return useQuery({
    queryKey: workOrdersQueryKeys.activity(workOrderId ?? ''),
    queryFn: () => workOrdersApi.listActivity(workOrderId!),
    enabled: Boolean(workOrderId),
  });
}

export function useRecentWorkOrderActivity(limit: number, offset: number) {
  return useQuery({
    queryKey: workOrdersQueryKeys.recentActivity(limit, offset),
    queryFn: () => workOrdersApi.getRecentActivity(limit, offset),
    placeholderData: (previousData) => previousData,
  });
}

function invalidateAfterWorkOrderChange(queryClient: QueryClient) {
  void queryClient.invalidateQueries({ queryKey: workOrdersQueryKeys.all });
}

export function useCreateWorkOrder(propertyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (values: WorkOrderFormValues) => workOrdersApi.create(propertyId, values),
    onSuccess: () => invalidateAfterWorkOrderChange(queryClient),
  });
}

export function useUpdateWorkOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: WorkOrderFormValues }) => workOrdersApi.update(id, values),
    onSuccess: () => invalidateAfterWorkOrderChange(queryClient),
  });
}

export function useDeleteWorkOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => workOrdersApi.delete(id),
    onSuccess: () => invalidateAfterWorkOrderChange(queryClient),
  });
}

export function useAddWorkOrderNote(workOrderId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ message, visibility }: { message: string; visibility: MaintenanceVisibility }) =>
      workOrdersApi.addNote(workOrderId, message, visibility),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: workOrdersQueryKeys.activity(workOrderId) });
    },
  });
}

export function useBulkUpdateWorkOrderStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ ids, status }: { ids: string[]; status: WorkOrderStatus }) => workOrdersApi.bulkUpdateStatus(ids, status),
    onSuccess: () => invalidateAfterWorkOrderChange(queryClient),
  });
}

export function useBulkReassignWorkOrders() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ ids, assignedTo }: { ids: string[]; assignedTo: string }) => workOrdersApi.bulkReassign(ids, assignedTo),
    onSuccess: () => invalidateAfterWorkOrderChange(queryClient),
  });
}
