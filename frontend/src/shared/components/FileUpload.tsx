import { useRef, useState } from 'react';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import AttachFileOutlined from '@mui/icons-material/AttachFileOutlined';
import { ApiError } from '@/api/client';
import { tokens } from '@/app/tokens';
import { attachmentsApi, ALLOWED_ATTACHMENT_TYPES, validateAttachmentFile } from '@/shared/lib/attachmentsApi';

interface FileUploadProps {
  label: string;
  /** Attachment id, or '' for none. */
  value: string;
  /** Name to show for a just-uploaded file; an existing attachment shows a generic label. */
  filename?: string;
  onChange: (attachmentId: string, filename: string) => void;
  disabled?: boolean;
  helperText?: string;
}

/**
 * Upload a receipt / photo / PDF straight to object storage and hand the
 * resulting attachment id to the form. Validation (type, 10 MB) runs
 * client-side for a fast message; the server and the storage policy
 * enforce the same limits regardless.
 */
export function FileUpload({ label, value, filename, onChange, disabled, helperText }: FileUploadProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleFile(file: File) {
    const problem = validateAttachmentFile(file);
    if (problem) {
      setError(problem);
      return;
    }
    setError(null);
    setBusy(true);
    try {
      const uploaded = await attachmentsApi.upload(file);
      onChange(uploaded.id, uploaded.filename);
    } catch (err) {
      setError(
        err instanceof ApiError && err.status === 503
          ? 'File uploads are not configured on this server.'
          : err instanceof Error
            ? err.message
            : 'The upload failed. Please try again.',
      );
    } finally {
      setBusy(false);
      if (inputRef.current) inputRef.current.value = '';
    }
  }

  async function view() {
    try {
      const { url } = await attachmentsApi.getUrl(value);
      window.open(url, '_blank', 'noopener,noreferrer');
    } catch {
      setError('Could not open the file.');
    }
  }

  return (
    <Box>
      <Typography sx={{ fontSize: 12.5, color: tokens.slate[600], mb: 0.75 }}>{label}</Typography>
      <input
        ref={inputRef}
        type="file"
        hidden
        accept={ALLOWED_ATTACHMENT_TYPES.join(',')}
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) void handleFile(file);
        }}
      />
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap', rowGap: 1 }}>
        {value ? (
          <>
            <Chip icon={<AttachFileOutlined />} label={filename || 'Attached file'} onClick={() => void view()} size="small" />
            {!disabled && (
              <>
                <Button size="small" variant="text" onClick={() => inputRef.current?.click()} disabled={busy}>
                  Replace
                </Button>
                <Button size="small" variant="text" color="error" onClick={() => onChange('', '')} disabled={busy}>
                  Remove
                </Button>
              </>
            )}
          </>
        ) : (
          <Button
            size="small"
            variant="outlined"
            disabled={disabled || busy}
            onClick={() => inputRef.current?.click()}
            startIcon={busy ? <CircularProgress size={14} /> : <AttachFileOutlined />}
            sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}
          >
            {busy ? 'Uploading…' : 'Choose file'}
          </Button>
        )}
      </Stack>
      {(error || helperText) && (
        <Typography sx={{ fontSize: 12, mt: 0.5, color: error ? tokens.error : tokens.slate[500] }}>{error ?? helperText}</Typography>
      )}
    </Box>
  );
}
