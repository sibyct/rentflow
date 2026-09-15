import { useRef, useState } from 'react';
import { Controller, type Control } from 'react-hook-form';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import CloseOutlined from '@mui/icons-material/CloseOutlined';
import CloudUploadOutlined from '@mui/icons-material/CloudUploadOutlined';
import ElevatorOutlined from '@mui/icons-material/ElevatorOutlined';
import EvStationOutlined from '@mui/icons-material/EvStationOutlined';
import FitnessCenterOutlined from '@mui/icons-material/FitnessCenterOutlined';
import InsertDriveFileOutlined from '@mui/icons-material/InsertDriveFileOutlined';
import Inventory2Outlined from '@mui/icons-material/Inventory2Outlined';
import LocalLaundryServiceOutlined from '@mui/icons-material/LocalLaundryServiceOutlined';
import LocalParkingOutlined from '@mui/icons-material/LocalParkingOutlined';
import PetsOutlined from '@mui/icons-material/PetsOutlined';
import PoolOutlined from '@mui/icons-material/PoolOutlined';
import { tokens } from '@/app/tokens';
import { AMENITIES } from '../../mock/propertyRows';
import type { AddPropertyFormValues } from '../../schemas/addPropertySchema';

const AMENITY_ICONS: Record<(typeof AMENITIES)[number], React.ComponentType<{ sx?: object }>> = {
  Parking: LocalParkingOutlined,
  Laundry: LocalLaundryServiceOutlined,
  Pool: PoolOutlined,
  Elevator: ElevatorOutlined,
  'Pet-friendly': PetsOutlined,
  Gym: FitnessCenterOutlined,
  Storage: Inventory2Outlined,
  'EV charging': EvStationOutlined,
};

export interface UploadedFile {
  id: string;
  file: File;
  /** Only set for images — revoked on removal, see AddPropertyModal. */
  previewUrl?: string;
}

interface AmenitiesMediaStepProps {
  control: Control<AddPropertyFormValues>;
  files: UploadedFile[];
  onAddFiles: (fileList: FileList | null) => void;
  onRemoveFile: (id: string) => void;
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export function AmenitiesMediaStep({ control, files, onAddFiles, onRemoveFile }: AmenitiesMediaStepProps) {
  const [dragOver, setDragOver] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  return (
    <Stack spacing={3.5}>
      <Box>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: 'text.secondary', mb: 1 }}>
          Amenities{' '}
          <Box component="span" sx={{ color: tokens.slate[400], fontWeight: 500 }}>
            (optional)
          </Box>
        </Typography>
        <Controller
          name="amenities"
          control={control}
          render={({ field }) => (
            // Wraps to new lines rather than overflowing horizontally —
            // this is a grid of chips, not a single scrollable row.
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
              {AMENITIES.map((a) => {
                const on = field.value.includes(a);
                const Icon = AMENITY_ICONS[a];
                return (
                  <Chip
                    key={a}
                    icon={<Icon sx={{ fontSize: 15 }} />}
                    label={a}
                    clickable
                    onClick={() => field.onChange(on ? field.value.filter((v) => v !== a) : [...field.value, a])}
                    variant={on ? 'filled' : 'outlined'}
                    color={on ? 'primary' : 'default'}
                    size="small"
                  />
                );
              })}
            </Box>
          )}
        />
      </Box>

      <Box>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: 'text.secondary', mb: 1 }}>
          Photos &amp; documents{' '}
          <Box component="span" sx={{ color: tokens.slate[400], fontWeight: 500 }}>
            (optional)
          </Box>
        </Typography>

        <Box
          onClick={() => inputRef.current?.click()}
          onDragOver={(e) => {
            e.preventDefault();
            setDragOver(true);
          }}
          onDragLeave={() => setDragOver(false)}
          onDrop={(e) => {
            e.preventDefault();
            setDragOver(false);
            onAddFiles(e.dataTransfer.files);
          }}
          sx={{
            border: `1.5px dashed ${dragOver ? tokens.azure[500] : tokens.slate[200]}`,
            bgcolor: dragOver ? tokens.azure[50] : tokens.slate[50],
            borderRadius: `${tokens.radiusCard}px`,
            p: 3,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: 0.5,
            textAlign: 'center',
            cursor: 'pointer',
          }}
        >
          <CloudUploadOutlined sx={{ fontSize: 26, color: tokens.slate[400] }} />
          <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>
            Drag photos &amp; documents here, or click to browse
          </Typography>
          <Typography sx={{ fontSize: 11.5, color: tokens.slate[400] }}>Images, PDFs — up to 10 files</Typography>
        </Box>
        <input
          ref={inputRef}
          type="file"
          accept="image/*,application/pdf"
          multiple
          hidden
          onChange={(e) => {
            onAddFiles(e.target.files);
            e.target.value = '';
          }}
        />

        {files.length > 0 && (
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1.25, mt: 1.5 }}>
            {files.map((f) =>
              f.previewUrl ? (
                <Box
                  key={f.id}
                  sx={{ position: 'relative', width: 72, height: 72, borderRadius: 1.5, overflow: 'hidden', border: `1px solid ${tokens.slate[200]}` }}
                >
                  <Box component="img" src={f.previewUrl} alt={f.file.name} sx={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }} />
                  <IconButton
                    size="small"
                    onClick={() => onRemoveFile(f.id)}
                    aria-label={`Remove ${f.file.name}`}
                    sx={{ position: 'absolute', top: 3, right: 3, width: 20, height: 20, bgcolor: 'rgba(15,17,22,0.65)', color: '#fff', '&:hover': { bgcolor: 'rgba(15,17,22,0.85)' } }}
                  >
                    <CloseOutlined sx={{ fontSize: 13 }} />
                  </IconButton>
                </Box>
              ) : (
                <Stack
                  key={f.id}
                  direction="row"
                  spacing={1}
                  sx={{ alignItems: 'center', border: `1px solid ${tokens.slate[200]}`, borderRadius: `${tokens.radiusControl}px`, p: 1, bgcolor: tokens.slate[50], maxWidth: 220 }}
                >
                  <InsertDriveFileOutlined sx={{ fontSize: 18, color: tokens.azure[600], flexShrink: 0 }} />
                  <Box sx={{ minWidth: 0 }}>
                    <Typography sx={{ fontSize: 12.5, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                      {f.file.name}
                    </Typography>
                    <Typography sx={{ fontSize: 11, color: tokens.slate[500], fontFamily: tokens.fontMono }}>{formatFileSize(f.file.size)}</Typography>
                  </Box>
                  <IconButton size="small" onClick={() => onRemoveFile(f.id)} aria-label={`Remove ${f.file.name}`} sx={{ ml: 'auto' }}>
                    <CloseOutlined sx={{ fontSize: 14 }} />
                  </IconButton>
                </Stack>
              ),
            )}
          </Box>
        )}
      </Box>
    </Stack>
  );
}
