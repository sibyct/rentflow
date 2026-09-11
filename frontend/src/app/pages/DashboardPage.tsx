import Stack from '@mui/material/Stack';
import { MetricCard, PlaceholderPage } from '@/shared/components';

// Static demo figures: there's no metrics/reporting backend yet. Kept
// here rather than invented server-side so it's obvious this is
// placeholder content for the shell, not real portfolio data.
export function DashboardPage() {
  return (
    <>
      <Stack direction="row" spacing={2} sx={{ flexWrap: 'wrap', mb: 3 }}>
        <MetricCard label="Collected this month" value="$114,220" delta="96.5% of billed · +1.2 pts" />
        <MetricCard label="Occupancy" value="94%" delta="45 of 48 units leased" />
        <MetricCard label="Outstanding balance" value="$4,180" delta="2 accounts past due" deltaTone="error" />
        <MetricCard label="Open work orders" value="7" delta="Median 2.1 days to close" />
      </Stack>

      <PlaceholderPage label="Dashboard" />
    </>
  );
}
