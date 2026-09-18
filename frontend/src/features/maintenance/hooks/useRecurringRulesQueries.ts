import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { recurringRulesApi } from '../api/recurringRulesApi';
import { workOrdersQueryKeys } from './useWorkOrdersQueries';
import type { RecurringRuleFormValues } from '../schemas/recurringRuleSchema';

export const recurringRulesQueryKeys = {
  all: ['recurringRules'] as const,
};

export function useRecurringRules() {
  return useQuery({
    queryKey: recurringRulesQueryKeys.all,
    queryFn: () => recurringRulesApi.list(),
  });
}

export function useCreateRecurringRule(propertyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (values: RecurringRuleFormValues) => recurringRulesApi.create(propertyId, values),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: recurringRulesQueryKeys.all });
    },
  });
}

export function useUpdateRecurringRule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: RecurringRuleFormValues }) => recurringRulesApi.update(id, values),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: recurringRulesQueryKeys.all });
    },
  });
}

export function useSetRecurringRuleActive() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, active }: { id: string; active: boolean }) => recurringRulesApi.setActive(id, active),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: recurringRulesQueryKeys.all });
    },
  });
}

export function useDeleteRecurringRule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => recurringRulesApi.delete(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: recurringRulesQueryKeys.all });
    },
  });
}

export function useGenerateWorkOrderNow() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => recurringRulesApi.generateNow(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: recurringRulesQueryKeys.all });
      // A newly generated work order needs the Work Orders list/summary
      // to reflect it too, not just this rule's own next_due_date.
      void queryClient.invalidateQueries({ queryKey: workOrdersQueryKeys.all });
    },
  });
}
