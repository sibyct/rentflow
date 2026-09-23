import { useNavigate } from 'react-router-dom';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useQueryClient } from '@tanstack/react-query';
import AddOutlined from '@mui/icons-material/AddOutlined';
import BusinessOutlined from '@mui/icons-material/BusinessOutlined';
import RefreshOutlined from '@mui/icons-material/RefreshOutlined';
import { tokens } from '@/app/tokens';
import { useProperties, propertiesQueryKeys } from '@/features/properties/hooks/usePropertiesQueries';
import { leasesQueryKeys } from '@/features/leases/hooks/useLeasesQueries';
import { unitsQueryKeys } from '@/features/units/hooks/useUnitsQueries';
import { workOrdersQueryKeys } from '@/features/maintenance/hooks/useWorkOrdersQueries';
import { vendorsQueryKeys } from '@/features/vendors/hooks/useVendorsQueries';
import { EmptyState, LoadingSpinner } from '@/shared/components';
import { ActivityFeed } from './ActivityFeed';
import { ComplianceAlertsCard } from './ComplianceAlertsCard';
import { ExpiringLeasesList } from './ExpiringLeasesList';
import { IncomeExpenseChart } from '@/features/accounting/components/IncomeExpenseChart';
import { KpiStrip } from './KpiStrip';
import { MaintenanceSummaryCard } from './MaintenanceSummaryCard';
import { VacantUnitsList } from './VacantUnitsList';

/**
 * The home screen: a read-only, glanceable overview. Every widget
 * fetches independently (each row/card owns its own query), so the
 * page renders progressively rather than blocking on the slowest
 * section — no single query gates the whole page the way the old
 * static placeholder didn't need to worry about at all.
 */
export function DashboardScreen() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  // Only used to decide the zero-properties empty state — every widget
  // below fetches its own data independently regardless.
  const { data: propertiesData, isLoading: propertiesLoading } = useProperties({ limit: 1, offset: 0 });

  function refreshAll() {
    void queryClient.invalidateQueries({ queryKey: propertiesQueryKeys.all });
    void queryClient.invalidateQueries({ queryKey: leasesQueryKeys.all });
    void queryClient.invalidateQueries({ queryKey: unitsQueryKeys.all });
    void queryClient.invalidateQueries({ queryKey: workOrdersQueryKeys.all });
    void queryClient.invalidateQueries({ queryKey: vendorsQueryKeys.all });
  }

  if (propertiesLoading) {
    return (
      <Box sx={{ py: 8 }}>
        <LoadingSpinner label="Loading dashboard…" />
      </Box>
    );
  }

  if ((propertiesData?.total ?? 0) === 0) {
    return (
      <EmptyState
        icon={BusinessOutlined}
        title="Get started"
        description="Add your first property to start seeing occupancy, leases, and maintenance activity here."
        action={
          <Button variant="contained" startIcon={<AddOutlined />} onClick={() => navigate('/properties')}>
            Add a property
          </Button>
        }
      />
    );
  }

  return (
    <Box>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
        <Typography variant="h3">Dashboard</Typography>
        <Tooltip title="Refresh">
          <IconButton onClick={refreshAll} aria-label="Refresh dashboard data">
            <RefreshOutlined fontSize="small" />
          </IconButton>
        </Tooltip>
      </Stack>

      {/* Row 1 */}
      <KpiStrip />

      {/* Row 2: ~60/40 split */}
      <Stack direction={{ xs: 'column', lg: 'row' }} spacing={2.5} sx={{ mb: 2.5, alignItems: 'stretch' }}>
        <Box sx={{ flex: '3 1 0', minWidth: 0 }}>
          <IncomeExpenseChart />
        </Box>
        <Stack spacing={2.5} sx={{ flex: '2 1 0', minWidth: 0 }}>
          <ExpiringLeasesList />
          <VacantUnitsList />
        </Stack>
      </Stack>

      {/* Row 3: 50/50 split */}
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2.5} sx={{ mb: 2.5, alignItems: 'stretch' }}>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <MaintenanceSummaryCard />
        </Box>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <ComplianceAlertsCard />
        </Box>
      </Stack>

      {/* Row 4 */}
      <ActivityFeed />

      <Typography sx={{ fontSize: 11.5, color: tokens.slate[400], mt: 2 }}>
        Money figures are cash basis, from the accounting ledger — payments actually received and paid out.
      </Typography>
    </Box>
  );
}
