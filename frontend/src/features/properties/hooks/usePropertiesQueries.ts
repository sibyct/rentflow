import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { propertiesApi, type PropertyListParams } from '../api/propertiesApi';
import type { PropertyRowStatus } from '../mock/propertyRows';
import type { AddPropertyFormValues } from '../schemas/addPropertySchema';

export const propertiesQueryKeys = {
  all: ['properties'] as const,
  list: (params: PropertyListParams) => [...propertiesQueryKeys.all, 'list', params] as const,
  detail: (id: string) => [...propertiesQueryKeys.all, 'detail', id] as const,
};

export function useProperties(params: PropertyListParams) {
  return useQuery({
    queryKey: propertiesQueryKeys.list(params),
    queryFn: () => propertiesApi.list(params),
    // Keep showing the previous page's rows while the next page/filter
    // loads, instead of flashing the table's empty/loading state on
    // every filter change or page click.
    placeholderData: (previousData) => previousData,
  });
}

/**
 * A cheap (limit=1) count for a given filter subset, used only to
 * explain *why* the main list came back empty — e.g. whether any
 * properties exist at all, or whether one particular filter (typically
 * the default "Active" status) is the one hiding otherwise-real rows.
 * Callers should keep this disabled until the main list is actually
 * empty; there's no reason to pay for a second request otherwise.
 */
export function usePropertiesTotalCount(
  params: Pick<PropertyListParams, 'search' | 'type' | 'status'>,
  options: { enabled: boolean },
) {
  return useQuery({
    queryKey: [...propertiesQueryKeys.all, 'count', params],
    queryFn: () => propertiesApi.list({ ...params, limit: 1, offset: 0 }),
    select: (result) => result.total,
    enabled: options.enabled,
  });
}

/** Fetches one property's full detail — the view/edit views. */
export function useProperty(id: string | undefined) {
  return useQuery({
    queryKey: propertiesQueryKeys.detail(id ?? ''),
    queryFn: () => propertiesApi.get(id!),
    enabled: Boolean(id),
  });
}

export function useCreateProperty() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (values: AddPropertyFormValues) => propertiesApi.create(values),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: propertiesQueryKeys.all });
    },
  });
}

export function useUpdateProperty() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: AddPropertyFormValues }) => propertiesApi.update(id, values),
    onSuccess: () => {
      // Matches the 'properties' prefix on both the list and every
      // detail query, so the edited property's own detail view (if
      // open) and the list it appears in both pick up the change.
      void queryClient.invalidateQueries({ queryKey: propertiesQueryKeys.all });
    },
  });
}

export function useUpdatePropertyStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ ids, status }: { ids: string[]; status: PropertyRowStatus }) => propertiesApi.updateStatus(ids, status),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: propertiesQueryKeys.all });
    },
  });
}
