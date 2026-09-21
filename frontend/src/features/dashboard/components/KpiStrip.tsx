import { useNavigate } from 'react-router-dom';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import { useProperties } from '@/features/properties/hooks/usePropertiesQueries';
import { useLeasesPortfolio } from '@/features/leases/hooks/useLeasesQueries';
import { useWorkOrderSummary } from '@/features/maintenance/hooks/useWorkOrdersQueries';
import { MetricCard } from '@/shared/components';

const currency = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });

/**
 * 5 KPI cards. 3 are real (Occupancy, Leases Expiring, Open Maintenance)
 * — 2 (Collected This Month, Outstanding Balance) are static demo
 * values, clearly labeled: there's no payments/invoicing feature in
 * this backend to read real figures from (see PropertyUnitStats'
 * TotalCollected doc comment — it's a rent-roll approximation, not
 * receipts, and there's no arrears/outstanding-balance concept at
 * all). Matches this app's no-fabricated-data convention everywhere
 * else, and the honesty this same page's static placeholder had before
 * this Dashboard existed.
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
      <Tooltip title="Demo data — there's no payments feature yet to read real figures from">
        <span>
          <MetricCard label="Collected This Month" value="$114,220" delta="Demo data · 96.5% of billed" />
        </span>
      </Tooltip>
      <Tooltip title="Demo data — there's no payments feature yet to read real figures from">
        <span>
          <MetricCard label="Outstanding Balance" value={currency.format(4180)} delta="Demo data · 2 accounts past due" deltaTone="error" />
        </span>
      </Tooltip>
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
