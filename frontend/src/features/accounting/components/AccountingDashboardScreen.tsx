import { Link as RouterLink, useNavigate } from 'react-router-dom';
import Box from '@mui/material/Box';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import ArrowForwardOutlined from '@mui/icons-material/ArrowForwardOutlined';
import DescriptionOutlined from '@mui/icons-material/DescriptionOutlined';
import ReceiptLongOutlined from '@mui/icons-material/ReceiptLongOutlined';
import RequestQuoteOutlined from '@mui/icons-material/RequestQuoteOutlined';
import { tokens } from '@/app/tokens';
import { ErrorMessage, MetricCard } from '@/shared/components';
import { formatMoneyWhole } from '@/shared/lib/format';
import { useAccountingDashboard } from '../hooks/useAccountingQueries';
import { IncomeExpenseChart } from './IncomeExpenseChart';

const QUICK_LINKS = [
  { label: 'Rent Roll', description: 'Who has paid, who is late', to: '/accounting/rent-roll', icon: RequestQuoteOutlined },
  { label: 'Expenses', description: 'Bills, repairs and recurring costs', to: '/accounting/expenses', icon: ReceiptLongOutlined },
  { label: 'Charges', description: 'One-off charges to tenants', to: '/accounting/charges', icon: DescriptionOutlined },
];

/** /accounting — the money at a glance: this month's KPIs, the income/expense trend, and shortcuts into each ledger view. */
export function AccountingDashboardScreen() {
  const navigate = useNavigate();
  // The KPIs are always this calendar month; only the chart has a period
  // toggle. Any period returns the same totals, so the default is reused
  // (and shared with the chart's initial query).
  const { data, isLoading, isError, error, refetch } = useAccountingDashboard('6months');

  if (isError) return <ErrorMessage error={error} onRetry={() => void refetch()} />;

  const collectedPct = data && data.expectedCents > 0 ? Math.round((data.collectedCents / data.expectedCents) * 100) : null;

  return (
    <Box>
      <Stack direction="row" spacing={2} sx={{ flexWrap: 'wrap', rowGap: 2, mb: 3 }}>
        <MetricCard
          label="Collected This Month"
          loading={isLoading}
          value={formatMoneyWhole(data?.collectedCents ?? 0)}
          delta={
            collectedPct === null
              ? 'Nothing billed yet'
              : `${collectedPct}% of ${formatMoneyWhole(data?.expectedCents ?? 0)} expected`
          }
          deltaTone={collectedPct !== null && collectedPct >= 90 ? 'success' : 'neutral'}
          onClick={() => navigate('/accounting/rent-roll')}
        />
        <MetricCard
          label="Outstanding Balance"
          loading={isLoading}
          value={formatMoneyWhole(data?.outstandingCents ?? 0)}
          delta={data && data.lateRentCount > 0 ? `${data.lateRentCount} late` : 'Nothing late'}
          deltaTone={data && data.lateRentCount > 0 ? 'error' : 'success'}
          onClick={() => navigate('/accounting/rent-roll?status=late')}
        />
        <MetricCard
          label="Net Income This Month"
          loading={isLoading}
          value={formatMoneyWhole(data?.netIncomeCents ?? 0)}
          delta={`${formatMoneyWhole(data?.expensesPaidCents ?? 0)} paid out`}
          deltaTone={data && data.netIncomeCents < 0 ? 'error' : 'neutral'}
          onClick={() => navigate('/accounting/expenses')}
        />
        <MetricCard
          label="Upcoming Expenses Due"
          loading={isLoading}
          value={formatMoneyWhole(data?.upcomingExpenseCents ?? 0)}
          delta={
            data && data.overdueExpenseCount > 0
              ? `${data.overdueExpenseCount} overdue`
              : `${data?.upcomingExpenseCount ?? 0} in the next 30 days`
          }
          deltaTone={data && data.overdueExpenseCount > 0 ? 'error' : 'neutral'}
          onClick={() => navigate('/accounting/expenses?status=unpaid')}
        />
      </Stack>

      <Box sx={{ mb: 3 }}>
        <IncomeExpenseChart />
      </Box>

      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', sm: 'repeat(3, 1fr)' } }}>
        {QUICK_LINKS.map(({ label, description, to, icon: Icon }) => (
          <Paper
            key={to}
            variant="outlined"
            component={RouterLink}
            to={to}
            sx={{
              p: 2,
              display: 'flex',
              alignItems: 'center',
              gap: 1.5,
              textDecoration: 'none',
              color: 'inherit',
              '&:hover': { boxShadow: tokens.shadowPopover },
            }}
          >
            <Icon sx={{ color: tokens.brand.green }} />
            <Box sx={{ flex: 1, minWidth: 0 }}>
              <Typography sx={{ fontSize: 14, fontWeight: 600, color: tokens.slate[900] }}>{label}</Typography>
              <Typography sx={{ fontSize: 12.5, color: tokens.slate[500] }}>{description}</Typography>
            </Box>
            <ArrowForwardOutlined sx={{ fontSize: 18, color: tokens.slate[400] }} />
          </Paper>
        ))}
      </Box>
    </Box>
  );
}
