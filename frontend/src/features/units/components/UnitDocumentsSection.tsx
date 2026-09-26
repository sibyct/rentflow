import { useRef, useState } from 'react';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import IconButton from '@mui/material/IconButton';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import DeleteOutlineOutlined from '@mui/icons-material/DeleteOutlineOutlined';
import DescriptionOutlined from '@mui/icons-material/DescriptionOutlined';
import DownloadOutlined from '@mui/icons-material/DownloadOutlined';
import UploadFileOutlined from '@mui/icons-material/UploadFileOutlined';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import { attachmentsApi, ALLOWED_ATTACHMENT_TYPES, validateAttachmentFile } from '@/shared/lib/attachmentsApi';
import { useAddUnitDocument, useDeleteUnitDocument, useUnitDocuments } from '../hooks/useUnitDocumentsQueries';
import { UNIT_DOCUMENT_CATEGORIES, UNIT_DOCUMENT_CATEGORY_LABELS, type UnitDocument, type UnitDocumentCategory } from '../types';

const dateFormatter = new Intl.DateTimeFormat('en-US', { dateStyle: 'medium' });

function formatDate(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : dateFormatter.format(d);
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

interface UnitDocumentsSectionProps {
  unitId: string;
}

/**
 * Unit-level Documents tab (FR6-FR9 of the Unit Detail redesign) — a
 * growing list, so this calls attachmentsApi.upload directly rather
 * than reusing FileUpload (built for one optional attachment slot).
 * Only ever lists unit_documents rows: no lease-linked attachments
 * exist today for FR9 to worry about duplicating.
 */
export function UnitDocumentsSection({ unitId }: UnitDocumentsSectionProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [pendingFile, setPendingFile] = useState<File | null>(null);
  const [category, setCategory] = useState<UnitDocumentCategory>('other');
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<UnitDocument | null>(null);
  const [toast, setToast] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);

  const { data: documents, isLoading } = useUnitDocuments(unitId);
  const addDocument = useAddUnitDocument(unitId);
  const deleteDocument = useDeleteUnitDocument(unitId);

  function handleFileSelected(file: File) {
    const problem = validateAttachmentFile(file);
    if (problem) {
      setUploadError(problem);
      return;
    }
    setUploadError(null);
    setPendingFile(file);
  }

  async function confirmUpload() {
    if (!pendingFile) return;
    setUploading(true);
    setUploadError(null);
    try {
      const uploaded = await attachmentsApi.upload(pendingFile);
      await addDocument.mutateAsync({ attachmentId: uploaded.id, category });
      setToast({ message: 'Document added', severity: 'success' });
      setPendingFile(null);
      setCategory('other');
    } catch (err) {
      setUploadError(
        err instanceof ApiError && err.status === 503
          ? 'File uploads are not configured on this server.'
          : err instanceof Error
            ? err.message
            : 'The upload failed. Please try again.',
      );
    } finally {
      setUploading(false);
      if (inputRef.current) inputRef.current.value = '';
    }
  }

  async function viewDocument(d: UnitDocument) {
    try {
      const { url } = await attachmentsApi.getUrl(d.attachmentId);
      window.open(url, '_blank', 'noopener,noreferrer');
    } catch {
      setToast({ message: 'Could not open the file.', severity: 'error' });
    }
  }

  const docs = documents ?? [];

  return (
    <Paper variant="outlined" sx={{ p: 3 }}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 2, flexWrap: 'wrap', gap: 1 }}>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>Documents</Typography>
        <input
          ref={inputRef}
          type="file"
          hidden
          accept={ALLOWED_ATTACHMENT_TYPES.join(',')}
          onChange={(e) => {
            const file = e.target.files?.[0];
            if (file) handleFileSelected(file);
          }}
        />
        <Button size="small" variant="contained" startIcon={<UploadFileOutlined />} onClick={() => inputRef.current?.click()}>
          Upload Document
        </Button>
      </Stack>

      {isLoading ? (
        <Box sx={{ py: 4, textAlign: 'center' }}>
          <CircularProgress size={22} />
        </Box>
      ) : docs.length === 0 ? (
        <Typography sx={{ fontSize: 13.5, color: tokens.slate[400], py: 3, textAlign: 'center' }}>
          No documents uploaded yet.
        </Typography>
      ) : (
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>File</TableCell>
              <TableCell>Category</TableCell>
              <TableCell>Uploaded by</TableCell>
              <TableCell>Date</TableCell>
              <TableCell align="right">Size</TableCell>
              <TableCell align="right" />
            </TableRow>
          </TableHead>
          <TableBody>
            {docs.map((d) => (
              <TableRow key={d.id} hover>
                <TableCell>
                  <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                    <DescriptionOutlined sx={{ fontSize: 17, color: tokens.slate[400] }} />
                    <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>{d.filename}</Typography>
                  </Stack>
                </TableCell>
                <TableCell>
                  <Chip label={UNIT_DOCUMENT_CATEGORY_LABELS[d.category]} size="small" variant="outlined" />
                </TableCell>
                <TableCell>
                  <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>{d.uploadedByName}</Typography>
                </TableCell>
                <TableCell>
                  <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>{formatDate(d.createdAt)}</Typography>
                </TableCell>
                <TableCell align="right">
                  <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>{formatBytes(d.sizeBytes)}</Typography>
                </TableCell>
                <TableCell align="right">
                  <IconButton size="small" onClick={() => void viewDocument(d)} aria-label="Download">
                    <DownloadOutlined fontSize="small" />
                  </IconButton>
                  <IconButton size="small" onClick={() => setDeleteTarget(d)} aria-label="Delete">
                    <DeleteOutlineOutlined fontSize="small" />
                  </IconButton>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <Dialog open={Boolean(pendingFile)} onClose={() => (uploading ? undefined : setPendingFile(null))}>
        <DialogTitle>Upload document</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1, minWidth: 320 }}>
            <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>{pendingFile?.name}</Typography>
            <TextField
              select
              label="Category"
              value={category}
              onChange={(e) => setCategory(e.target.value as UnitDocumentCategory)}
              disabled={uploading}
              fullWidth
            >
              {UNIT_DOCUMENT_CATEGORIES.map((c) => (
                <MenuItem key={c} value={c}>
                  {UNIT_DOCUMENT_CATEGORY_LABELS[c]}
                </MenuItem>
              ))}
            </TextField>
            {uploadError && <Alert severity="error">{uploadError}</Alert>}
          </Stack>
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 0 }}>
          <Button variant="text" onClick={() => setPendingFile(null)} disabled={uploading} sx={{ color: tokens.slate[600] }}>
            Cancel
          </Button>
          <Button variant="contained" onClick={() => void confirmUpload()} disabled={uploading} startIcon={uploading ? <CircularProgress size={14} /> : undefined}>
            {uploading ? 'Uploading…' : 'Upload'}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={Boolean(deleteTarget)} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete this document?</DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13.5, color: tokens.slate[600] }}>
            {deleteTarget ? `"${deleteTarget.filename}"` : 'This document'} will be permanently removed. This can&apos;t be undone.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 0 }}>
          <Button variant="text" onClick={() => setDeleteTarget(null)} sx={{ color: tokens.slate[600] }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color="error"
            disabled={deleteDocument.isPending}
            onClick={() => {
              if (!deleteTarget) return;
              deleteDocument.mutate(deleteTarget.id, {
                onSuccess: () => {
                  setToast({ message: 'Document deleted', severity: 'success' });
                  setDeleteTarget(null);
                },
                onError: () => setToast({ message: 'Something went wrong. Please try again.', severity: 'error' }),
              });
            }}
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      <Snackbar open={Boolean(toast)} autoHideDuration={3200} onClose={() => setToast(null)} anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}>
        {toast ? (
          <Alert onClose={() => setToast(null)} severity={toast.severity} variant="filled" sx={{ borderRadius: 999 }}>
            {toast.message}
          </Alert>
        ) : undefined}
      </Snackbar>
    </Paper>
  );
}
