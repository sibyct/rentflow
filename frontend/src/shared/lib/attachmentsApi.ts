import { apiClient } from '@/api/client';

// Direct-to-storage uploads: the API hands out a presigned POST, the
// browser sends the file straight to the bucket (the bytes never pass
// through the API), then the API confirms the object landed. Mirrors
// backend/internal/domain/attachment.go.

export const ALLOWED_ATTACHMENT_TYPES = ['image/jpeg', 'image/png', 'image/webp', 'application/pdf'];
export const MAX_ATTACHMENT_BYTES = 10 * 1024 * 1024;

interface PresignWire {
  attachment: { id: string; filename: string };
  upload_url: string;
  fields: Record<string, string>;
}

export interface UploadedAttachment {
  id: string;
  filename: string;
}

/** Returns a user-facing problem with the file, or null if it can be uploaded. */
export function validateAttachmentFile(file: File): string | null {
  if (!ALLOWED_ATTACHMENT_TYPES.includes(file.type)) return 'Only JPEG, PNG, WebP images and PDFs can be uploaded.';
  if (file.size === 0) return 'That file is empty.';
  if (file.size > MAX_ATTACHMENT_BYTES) return 'Files can be at most 10 MB.';
  return null;
}

export const attachmentsApi = {
  async upload(file: File): Promise<UploadedAttachment> {
    const problem = validateAttachmentFile(file);
    if (problem) throw new Error(problem);

    const presign = await apiClient.post<PresignWire>('/api/v1/attachments/presign', {
      filename: file.name,
      content_type: file.type,
      size_bytes: file.size,
    });

    // Presigned POST policy: every signed field first, the file part last.
    const form = new FormData();
    for (const [key, value] of Object.entries(presign.fields)) form.append(key, value);
    form.append('file', file);
    const res = await fetch(presign.upload_url, { method: 'POST', body: form });
    if (!res.ok) throw new Error('The upload failed. Please try again.');

    await apiClient.post(`/api/v1/attachments/${presign.attachment.id}/confirm`);
    return { id: presign.attachment.id, filename: presign.attachment.filename };
  },

  /** A short-lived download URL for an attachment the caller owns. */
  getUrl(id: string): Promise<{ url: string; filename: string }> {
    return apiClient.get(`/api/v1/attachments/${id}/url`);
  },
};
