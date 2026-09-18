import ButtonBase from '@mui/material/ButtonBase';
import Skeleton from '@mui/material/Skeleton';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { tokens } from '@/app/tokens';
import type { WorkOrderSummary } from '../types';

interface CounterDef {
  key: 'open' | 'overdue' | 'unassigned' | 'emergency';
  label: string;
  color: string;
}

const COUNTERS: CounterDef[] = [
  { key: 'open', label: 'Open', color: tokens.azure[600] },
  { key: 'overdue', label: 'Overdue', color: tokens.error },
  { key: 'unassigned', label: 'Unassigned', color: tokens.warningInk },
  { key: 'emergency', label: 'Emergency', color: tokens.error },
];

interface MaintenanceSummaryCountersProps {
  loading: boolean;
  summary: WorkOrderSummary | undefined;
  active: CounterDef['key'] | null;
  onSelect: (key: CounterDef['key'] | null) => void;
}

/** Clickable summary counters above the Work Orders table — clicking one toggles the matching filter (see MaintenanceScreen), clicking the active one again clears it. */
export function MaintenanceSummaryCounters({ loading, summary, active, onSelect }: MaintenanceSummaryCountersProps) {
  return (
    <Stack direction="row" spacing={1.5} sx={{ flexWrap: 'wrap', mb: 2 }}>
      {COUNTERS.map((c) => {
        const isActive = active === c.key;
        return (
          <ButtonBase
            key={c.key}
            onClick={() => onSelect(isActive ? null : c.key)}
            sx={{
              flex: '1 1 140px',
              minWidth: 140,
              borderRadius: `${tokens.radiusCard}px`,
              border: `1.5px solid ${isActive ? c.color : tokens.slate[200]}`,
              bgcolor: isActive ? `${c.color}14` : 'common.white',
              p: 2,
              alignItems: 'flex-start',
              flexDirection: 'column',
              textAlign: 'left',
              transition: 'border-color 120ms ease, background-color 120ms ease',
            }}
          >
            <Typography sx={{ fontSize: 11, fontWeight: 600, letterSpacing: '0.04em', color: tokens.slate[500], textTransform: 'uppercase', mb: 0.5 }}>
              {c.label}
            </Typography>
            {loading || !summary ? (
              <Skeleton width={36} height={28} />
            ) : (
              <Typography sx={{ fontSize: 22, fontWeight: 700, color: c.color, fontFamily: tokens.fontMono }}>
                {summary[c.key]}
              </Typography>
            )}
          </ButtonBase>
        );
      })}
    </Stack>
  );
}
