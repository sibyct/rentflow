import { useState } from 'react';
import { BarChart } from '@mui/x-charts/BarChart';
import Box from '@mui/material/Box';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import InfoOutlined from '@mui/icons-material/InfoOutlined';
import { tokens } from '@/app/tokens';

type Period = 'month' | '6months' | 'ytd';

const PERIOD_LABELS: Record<Period, string> = {
  month: 'This Month',
  '6months': 'Last 6 Months',
  ytd: 'YTD',
};

// Demo dataset — there's no payments/expense-tracking feature in this
// backend (see KpiStrip's identical caveat). Shaped as monthly series
// so switching periods has something to actually show, not as a claim
// about real portfolio performance.
const DEMO_DATA: Record<Period, { labels: string[]; income: number[]; expense: number[] }> = {
  month: { labels: ['Week 1', 'Week 2', 'Week 3', 'Week 4'], income: [28500, 29200, 27800, 28700], expense: [8200, 6100, 9400, 5300] },
  '6months': {
    labels: ['Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep'],
    income: [108000, 111500, 109800, 114200, 112900, 114220],
    expense: [31200, 24800, 38100, 22400, 29600, 27300],
  },
  ytd: {
    labels: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep'],
    income: [104200, 105800, 106900, 108000, 111500, 109800, 114200, 112900, 114220],
    expense: [26100, 19800, 33400, 31200, 24800, 38100, 22400, 29600, 27300],
  },
};

export function IncomeExpenseChart() {
  const [period, setPeriod] = useState<Period>('6months');
  const data = DEMO_DATA[period];

  return (
    <Paper variant="outlined" sx={{ p: 2.5, height: '100%' }}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 1, mb: 1.5 }}>
        <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center' }}>
          <Typography sx={{ fontSize: 13, fontWeight: 600, color: tokens.slate[700] }}>Income vs Expense</Typography>
          <Tooltip title="Demo data — there's no payments/expense-tracking feature yet to read real figures from">
            <InfoOutlined sx={{ fontSize: 15, color: tokens.slate[400] }} />
          </Tooltip>
        </Stack>
        <ToggleButtonGroup
          exclusive
          size="small"
          value={period}
          onChange={(_e, v: Period | null) => v && setPeriod(v)}
        >
          {(Object.keys(PERIOD_LABELS) as Period[]).map((p) => (
            <ToggleButton key={p} value={p} sx={{ textTransform: 'none', fontSize: 12, px: 1.5 }}>
              {PERIOD_LABELS[p]}
            </ToggleButton>
          ))}
        </ToggleButtonGroup>
      </Stack>

      <Box sx={{ width: '100%', overflowX: 'auto' }}>
        <BarChart
          height={260}
          series={[
            { data: data.income, label: 'Income', color: tokens.brand.green },
            { data: data.expense, label: 'Expense', color: tokens.error },
          ]}
          xAxis={[{ data: data.labels, scaleType: 'band' }]}
          margin={{ left: 60, right: 10, top: 10, bottom: 30 }}
        />
      </Box>
    </Paper>
  );
}
