import type { ReactNode } from 'react';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { tokens } from '@/app/tokens';

interface FormSectionProps {
  title: string;
  children: ReactNode;
}

/** A titled group of form fields — the small-caps heading used inside every drawer/dialog form. */
export function FormSection({ title, children }: FormSectionProps) {
  return (
    <Stack spacing={1.5}>
      <Typography
        sx={{
          fontSize: 11,
          fontWeight: 600,
          letterSpacing: '0.04em',
          color: tokens.slate[500],
          textTransform: 'uppercase',
        }}
      >
        {title}
      </Typography>
      {children}
    </Stack>
  );
}
