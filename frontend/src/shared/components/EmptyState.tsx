import type { ComponentType, ReactNode } from 'react';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import type { SvgIconProps } from '@mui/material/SvgIcon';
import { tokens } from '@/app/tokens';

export interface EmptyStateProps {
  icon: ComponentType<SvgIconProps>;
  title: string;
  description: string;
  action?: ReactNode;
}

/**
 * Compact empty-state block for a table, list, or panel's own content
 * area — not full-bleed like ErrorState (which owns the whole page/shell
 * area for a hard failure). Use it wherever a collection legitimately
 * has nothing to show: no data at all, or nothing matching the current
 * filters. The caller supplies its own wrapping (e.g. a TableRow/
 * TableCell with colSpan for a table body) since that's specific to
 * where the empty state is being rendered.
 */
export function EmptyState({ icon: Icon, title, description, action }: EmptyStateProps) {
  return (
    <Stack sx={{ alignItems: 'center', textAlign: 'center', py: 8, px: 3, gap: 1.5 }}>
      <Box
        sx={{
          width: 56,
          height: 56,
          borderRadius: '50%',
          bgcolor: tokens.azure[50],
          color: tokens.azure[600],
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <Icon sx={{ fontSize: 26 }} />
      </Box>
      <Typography variant="h5" sx={{ fontSize: 17 }}>
        {title}
      </Typography>
      <Typography sx={{ fontSize: 13.5, color: tokens.slate[500], maxWidth: 340 }}>{description}</Typography>
      {action}
    </Stack>
  );
}
