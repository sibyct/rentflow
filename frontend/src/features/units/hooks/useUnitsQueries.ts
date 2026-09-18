import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query';
import { propertiesQueryKeys } from '@/features/properties/hooks/usePropertiesQueries';
import { unitsApi, type BulkUnitRow, type UnitPortfolioListParams } from '../api/unitsApi';
import type { UnitFormValues } from '../schemas/unitSchema';

export const unitsQueryKeys = {
  all: ['units'] as const,
  byProperty: (propertyId: string) => [...unitsQueryKeys.all, 'property', propertyId] as const,
  detail: (id: string) => [...unitsQueryKeys.all, 'detail', id] as const,
  portfolio: (params: UnitPortfolioListParams) => [...unitsQueryKeys.all, 'portfolio', params] as const,
};

export function useUnits(propertyId: string | undefined) {
  return useQuery({
    queryKey: unitsQueryKeys.byProperty(propertyId ?? ''),
    queryFn: () => unitsApi.list(propertyId!),
    enabled: Boolean(propertyId),
  });
}

/** The portfolio-wide list behind the global /units page. */
export function useUnitsPortfolio(params: UnitPortfolioListParams) {
  return useQuery({
    queryKey: unitsQueryKeys.portfolio(params),
    queryFn: () => unitsApi.listForOwner(params),
    placeholderData: (previousData) => previousData,
  });
}

export function useUnit(unitId: string | undefined) {
  return useQuery({
    queryKey: unitsQueryKeys.detail(unitId ?? ''),
    queryFn: () => unitsApi.get(unitId!),
    enabled: Boolean(unitId),
  });
}

// A unit change invalidates every units query (byProperty, portfolio —
// both the "properties.owner_id" prefix they share), plus the parent
// property's own detail query: the property's stat cards (occupancy,
// collected, unit count) are computed server-side from its units (see
// UnitRepository.GetPropertyUnitStats), so that query is stale too, not
// just the units lists. A change made from the global /units page or
// the property-scoped section both need to be visible in the other.
function invalidateAfterUnitChange(queryClient: QueryClient, propertyId: string) {
  void queryClient.invalidateQueries({ queryKey: unitsQueryKeys.all });
  void queryClient.invalidateQueries({ queryKey: propertiesQueryKeys.detail(propertyId) });
}

export function useCreateUnit(propertyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (values: UnitFormValues) => unitsApi.create(propertyId, values),
    onSuccess: () => invalidateAfterUnitChange(queryClient, propertyId),
  });
}

export function useCreateUnitsBulk(propertyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (rows: BulkUnitRow[]) => unitsApi.createBulk(propertyId, rows),
    onSuccess: () => invalidateAfterUnitChange(queryClient, propertyId),
  });
}

export function useUpdateUnit(propertyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: UnitFormValues }) => unitsApi.update(id, values),
    onSuccess: (_data, vars) => {
      invalidateAfterUnitChange(queryClient, propertyId);
      void queryClient.invalidateQueries({ queryKey: unitsQueryKeys.detail(vars.id) });
    },
  });
}

export function useDeleteUnit(propertyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => unitsApi.delete(id),
    onSuccess: () => invalidateAfterUnitChange(queryClient, propertyId),
  });
}
