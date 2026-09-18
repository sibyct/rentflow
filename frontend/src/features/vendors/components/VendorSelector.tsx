import Autocomplete from '@mui/material/Autocomplete';
import Box from '@mui/material/Box';
import Rating from '@mui/material/Rating';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import StarOutlined from '@mui/icons-material/StarOutlined';
import { tokens } from '@/app/tokens';
import type { WorkOrderCategory } from '@/features/maintenance/types';
import { useVendorsForCategory } from '../hooks/useVendorsQueries';
import type { VendorRow } from '../types';

interface VendorSelectorProps {
  category: WorkOrderCategory | undefined;
  value: string;
  onChange: (vendor: VendorRow | null) => void;
  disabled?: boolean;
}

/**
 * Reusable vendor picker for the work order drawer's Assignment
 * section — active vendors whose trades include the selected category,
 * so choices narrow as soon as a category is picked. Shows each
 * vendor's rating and current open job count so a manager can balance
 * load, per the spec. Disabled with a hint until a category is chosen,
 * since ListForCategory has nothing to return before then.
 */
export function VendorSelector({ category, value, onChange, disabled }: VendorSelectorProps) {
  const { data: vendors, isLoading } = useVendorsForCategory(category);
  const options = vendors ?? [];
  const selected = options.find((v) => v.id === value) ?? null;

  return (
    <Autocomplete<VendorRow>
      options={options}
      loading={isLoading}
      disabled={disabled || !category}
      value={selected}
      onChange={(_e, next) => onChange(next)}
      getOptionLabel={(v) => v.companyName}
      isOptionEqualToValue={(a, b) => a.id === b.id}
      renderOption={(props, v) => (
        <Box component="li" {...props} key={v.id}>
          <Stack sx={{ width: '100%' }}>
            <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
              <Typography sx={{ fontSize: 13.5, fontWeight: 600 }}>{v.companyName}</Typography>
              {v.averageRating != null && (
                <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
                  <StarOutlined sx={{ fontSize: 14, color: '#F5A623' }} />
                  <Typography sx={{ fontSize: 12, color: tokens.slate[600] }}>{v.averageRating.toFixed(1)}</Typography>
                </Stack>
              )}
            </Stack>
            <Typography sx={{ fontSize: 11.5, color: tokens.slate[500] }}>
              {v.openWorkOrders} open job{v.openWorkOrders === 1 ? '' : 's'}
            </Typography>
          </Stack>
        </Box>
      )}
      renderInput={(params) => (
        <TextField
          {...params}
          label="Vendor"
          placeholder={category ? 'Search vendors…' : 'Pick a category first'}
          helperText={!category ? 'Choose a category above to see matching vendors' : undefined}
        />
      )}
    />
  );
}

export function VendorRatingInput({ value, onChange }: { value: number | null; onChange: (v: number | null) => void }) {
  return (
    <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
      <Typography sx={{ fontSize: 13, color: tokens.slate[600] }}>Rate this vendor</Typography>
      <Rating value={value} onChange={(_e, v) => onChange(v)} />
    </Stack>
  );
}
