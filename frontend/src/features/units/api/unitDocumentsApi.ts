import { apiClient } from '@/api/client';
import type { UnitDocument, UnitDocumentCategory } from '../types';

// Wire shape exactly as internal/transport/http/dto/unit_document_dto.go's
// UnitDocumentResponse serializes it — see propertiesApi.ts's PropertyWire
// for why this stays private to this file.
interface UnitDocumentWire {
  id: string;
  unit_id: string;
  attachment_id: string;
  category: string;
  uploaded_by: string;
  uploaded_by_name: string;
  filename: string;
  content_type: string;
  size_bytes: number;
  created_at: string;
}

function toUnitDocument(wire: UnitDocumentWire): UnitDocument {
  return {
    id: wire.id,
    unitId: wire.unit_id,
    attachmentId: wire.attachment_id,
    category: wire.category as UnitDocumentCategory,
    uploadedBy: wire.uploaded_by,
    uploadedByName: wire.uploaded_by_name,
    filename: wire.filename,
    contentType: wire.content_type,
    sizeBytes: wire.size_bytes,
    createdAt: wire.created_at,
  };
}

export const unitDocumentsApi = {
  list: (unitId: string): Promise<UnitDocument[]> =>
    apiClient.get<UnitDocumentWire[]>(`/api/v1/units/${unitId}/documents`).then((rows) => rows.map(toUnitDocument)),

  add: (unitId: string, attachmentId: string, category: UnitDocumentCategory): Promise<UnitDocument> =>
    apiClient
      .post<UnitDocumentWire>(`/api/v1/units/${unitId}/documents`, { attachment_id: attachmentId, category })
      .then(toUnitDocument),

  delete: (unitId: string, documentId: string): Promise<void> =>
    apiClient.delete<void>(`/api/v1/units/${unitId}/documents/${documentId}`),
};
