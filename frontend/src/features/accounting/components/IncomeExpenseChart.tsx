import { useState } from 'react';
import { BarChart } from '@mui/x-charts/BarChart';
import Box from '@mui/material/Box';
import Paper from '@mui/material/Paper';
import Skeleton from '@mui/material/Skeleton';
import Stack from '@mui/material/Stack';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import InfoOutlined from '@mui/icons-material/InfoOutlined';
import { tokens } from '@/app/tokens';
import { formatMoneyWhole, parseDate } from '@/shared/lib/format';
import { useAccountingDashboard } from '../hooks/useAccountingQueries';
import type { DashboardPeriod } from '../types';

const PERIOD_LABELS: Record<DashboardPeriod, string> = {
  month: 'This Month',
  '6months': 'Last 6 Months',
  ytd: 'YTD',
};

function bucketLabel(iso: string, period: DashboardPeriod): string {
  const d = parseDate(iso);
  if (!d) return iso;
  return period === 'month'
    ? d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
    : d.toLocaleDateString('en-US', { month: 'short' });
}

/**
 * Income vs Expense from the ledger, cash basis: income is rent and
 * charges actually received, expense is what was actually paid out. The
 * period toggle re-queries the backend (weekly buckets for "This Month",
 * monthly for the others) — nothing is computed client-side.
 */
export function IncomeExpenseChart() {
  const [period, setPeriod] = useState<DashboardPeriod>('6months');
  const { data, isLoading } = useAccountingDashboard(period);
  const series = data?.series ?? [];

  return (
    <Paper variant="outlined" sx={{ p: 2.5, height: '100%' }}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 1, mb: 1.5 }}>
        <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center' }}>
          <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>Income vs Expense</Typography>
          <Tooltip title="Cash basis: money actually received vs. money actually paid out, by the date of the payment.">
            <InfoOutlined sx={{ fontSize: 15, color: tokens.slate[400] }} />
          </Tooltip>
        </Stack>
        <ToggleButtonGroup exclusive size="small" value={period} onChange={(_e, v: DashboardPeriod | null) => v && setPeriod(v)}>
          {(Object.keys(PERIOD_LABELS) as DashboardPeriod[]).map((p) => (
            <ToggleButton key={p} value={p} sx={{ textTransform: 'none', fontSize: 12, px: 1.5 }}>
              {PERIOD_LABELS[p]}
            </ToggleButton>
          ))}
        </ToggleButtonGroup>
      </Stack>

      {isLoading ? (
        <Skeleton variant="rounded" height={260} />
      ) : (
        <Box sx={{ width: '100%' }}>
          <BarChart
            height={260}
            series={[
              { data: series.map((p) => p.incomeCents), label: 'Income', color: tokens.brand.green, valueFormatter: (v) => (v == null ? '' : formatMoneyWhole(v)) },
              { data: series.map((p) => p.expenseCents), label: 'Expense', color: tokens.error, valueFormatter: (v) => (v == null ? '' : formatMoneyWhole(v)) },
            ]}
            xAxis={[{ data: series.map((p) => bucketLabel(p.bucket, period)), scaleType: 'band' }]}
            yAxis={[{ valueFormatter: (v: number) => formatMoneyWhole(v) }]}
            margin={{ left: 70, right: 10, top: 10, bottom: 30 }}
          />
        </Box>
      )}
    </Paper>
  );
}
