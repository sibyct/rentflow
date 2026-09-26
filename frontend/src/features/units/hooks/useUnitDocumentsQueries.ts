import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { unitDocumentsApi } from '../api/unitDocumentsApi';
import type { UnitDocumentCategory } from '../types';

export const unitDocumentsQueryKeys = {
  all: ['unitDocuments'] as const,
  byUnit: (unitId: string) => [...unitDocumentsQueryKeys.all, unitId] as const,
};

export function useUnitDocuments(unitId: string) {
  return useQuery({
    queryKey: unitDocumentsQueryKeys.byUnit(unitId),
    queryFn: () => unitDocumentsApi.list(unitId),
  });
}

export function useAddUnitDocument(unitId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ attachmentId, category, relatedLeaseId }: { attachmentId: string; category: UnitDocumentCategory; relatedLeaseId?: string }) =>
      unitDocumentsApi.add(unitId, attachmentId, category, relatedLeaseId),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: unitDocumentsQueryKeys.byUnit(unitId) }),
  });
}

export function useDeleteUnitDocument(unitId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (documentId: string) => unitDocumentsApi.delete(unitId, documentId),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: unitDocumentsQueryKeys.byUnit(unitId) }),
  });
}
