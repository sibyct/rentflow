import Chip from '@mui/material/Chip';

export type StatusTone = 'success' | 'warning' | 'error' | 'info' | 'default';

interface StatusChipProps {
  label: string;
  tone: StatusTone;
  variant?: 'filled' | 'outlined';
}

/**
 * The green / amber / red status badge the accounting screens share
 * (Rent Roll, Expenses, Charges, Deposits, Statements). Each feature owns
 * its own `status → tone` map next to its types, same as the older
 * per-table chip maps; this is only the rendering half.
 */
export function StatusChip({ label, tone, variant = 'filled' }: StatusChipProps) {
  return <Chip size="small" label={label} color={tone} variant={variant} sx={{ fontWeight: 600 }} />;
}
