import { useNavigate } from 'react-router-dom';
import Stack from '@mui/material/Stack';
import { useAccountingDashboard } from '@/features/accounting/hooks/useAccountingQueries';
import { useProperties } from '@/features/properties/hooks/usePropertiesQueries';
import { useLeasesPortfolio } from '@/features/leases/hooks/useLeasesQueries';
import { useWorkOrderSummary } from '@/features/maintenance/hooks/useWorkOrdersQueries';
import { MetricCard } from '@/shared/components';
import { formatMoneyWhole } from '@/shared/lib/format';

/**
 * 5 KPI cards, every one real: occupancy, leases expiring and open
 * maintenance from their own modules, and Collected This Month /
 * Outstanding Balance from the accounting ledger (cash basis — money
 * actually received this calendar month, and everything billed and due
 * but still unpaid).
 */
export function KpiStrip() {
  const navigate = useNavigate();

  // Every property an owner has, capped at 100 — same convention used
  // for every other portfolio-wide picker/rollup in this codebase.
  // PropertyResponse already carries real per-property occupancy/unit
  // counts (WithUnitStats), so the portfolio totals below are summed
  // client-side rather than needing a new backend aggregation endpoint.
  const { data: propertiesData, isLoading: propertiesLoading } = useProperties({ limit: 100, offset: 0 });
  const properties = propertiesData?.properties ?? [];
  const totalUnits = properties.reduce((sum, p) => sum + p.unitCount, 0);
  const occupiedUnits = properties.reduce((sum, p) => sum + Math.round((p.occupancyPct / 100) * p.unitCount), 0);
  const blendedOccupancyPct = totalUnits > 0 ? Math.round((occupiedUnits / totalUnits) * 100) : 0;

  const { data: accounting, isLoading: accountingLoading } = useAccountingDashboard('6months');
  const collectedPct =
    accounting && accounting.expectedCents > 0 ? Math.round((accounting.collectedCents / accounting.expectedCents) * 100) : null;

  const { data: leasesData, isLoading: leasesLoading } = useLeasesPortfolio({
    status: 'expiring_soon',
    limit: 1,
    offset: 0,
  });
  const expiringLeasesCount = leasesData?.total ?? 0;

  const { data: summary, isLoading: summaryLoading } = useWorkOrderSummary();

  return (
    <Stack direction="row" spacing={2} sx={{ flexWrap: 'wrap', mb: 3 }}>
      <MetricCard
        label="Total Units / Occupancy"
        value={propertiesLoading ? '' : `${totalUnits} · ${blendedOccupancyPct}%`}
        delta={propertiesLoading ? '' : `${occupiedUnits} of ${totalUnits} occupied`}
        loading={propertiesLoading}
        onClick={() => navigate('/units')}
      />
      <MetricCard
        label="Collected This Month"
        value={formatMoneyWhole(accounting?.collectedCents ?? 0)}
        delta={collectedPct === null ? 'Nothing billed yet' : `${collectedPct}% of ${formatMoneyWhole(accounting?.expectedCents ?? 0)} billed`}
        deltaTone={collectedPct !== null && collectedPct >= 90 ? 'success' : 'neutral'}
        loading={accountingLoading}
        onClick={() => navigate('/accounting/rent-roll')}
      />
      <MetricCard
        label="Outstanding Balance"
        value={formatMoneyWhole(accounting?.outstandingCents ?? 0)}
        delta={accounting && accounting.lateRentCount > 0 ? `${accounting.lateRentCount} past due` : 'Nothing past due'}
        deltaTone={accounting && accounting.lateRentCount > 0 ? 'error' : 'neutral'}
        loading={accountingLoading}
        onClick={() => navigate('/accounting/rent-roll?status=late')}
      />
      <MetricCard
        label="Open Maintenance Requests"
        value={summaryLoading ? '' : String(summary?.open ?? 0)}
        delta={summaryLoading ? '' : summary && summary.emergency > 0 ? `${summary.emergency} emergency` : 'None emergency'}
        deltaTone={summary && summary.emergency > 0 ? 'error' : 'neutral'}
        loading={summaryLoading}
        onClick={() => navigate('/maintenance')}
      />
      <MetricCard
        label="Leases Expiring"
        value={leasesLoading ? '' : String(expiringLeasesCount)}
        delta="Next 30 days"
        loading={leasesLoading}
        onClick={() => navigate('/leases?status=expiring_soon')}
      />
    </Stack>
  );
}
