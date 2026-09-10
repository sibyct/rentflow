import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { propertiesApi, type ListPropertiesParams } from '../api/propertiesApi';
import type { CreatePropertyInput, UpdatePropertyInput } from '../types';

export const propertyKeys = {
  all: ['properties'] as const,
  lists: () => [...propertyKeys.all, 'list'] as const,
  list: (params: ListPropertiesParams) => [...propertyKeys.lists(), params] as const,
  details: () => [...propertyKeys.all, 'detail'] as const,
  detail: (id: string) => [...propertyKeys.details(), id] as const,
};

export function useProperties(params: ListPropertiesParams = {}) {
  return useQuery({
    queryKey: propertyKeys.list(params),
    queryFn: () => propertiesApi.list(params),
  });
}

export function useProperty(id: string) {
  return useQuery({
    queryKey: propertyKeys.detail(id),
    queryFn: () => propertiesApi.get(id),
    enabled: Boolean(id),
  });
}

export function useCreateProperty() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreatePropertyInput) => propertiesApi.create(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: propertyKeys.lists() });
    },
  });
}

export function useUpdateProperty(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdatePropertyInput) => propertiesApi.update(id, input),
    onSuccess: (updated) => {
      queryClient.setQueryData(propertyKeys.detail(id), updated);
      void queryClient.invalidateQueries({ queryKey: propertyKeys.lists() });
    },
  });
}

export function useDeleteProperty() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => propertiesApi.delete(id),
    onSuccess: (_data, id) => {
      queryClient.removeQueries({ queryKey: propertyKeys.detail(id) });
      void queryClient.invalidateQueries({ queryKey: propertyKeys.lists() });
    },
  });
}
