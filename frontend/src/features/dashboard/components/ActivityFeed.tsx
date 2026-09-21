import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import BuildOutlined from '@mui/icons-material/BuildOutlined';
import DescriptionOutlined from '@mui/icons-material/DescriptionOutlined';
import StorefrontOutlined from '@mui/icons-material/StorefrontOutlined';
import { tokens } from '@/app/tokens';
import { useRecentWorkOrderActivity } from '@/features/maintenance/hooks/useWorkOrdersQueries';
import { useLeasesPortfolio } from '@/features/leases/hooks/useLeasesQueries';
import { useVendors } from '@/features/vendors/hooks/useVendorsQueries';
import { EmptyState, LoadingSpinner } from '@/shared/components';
import { relativeTime } from '@/shared/lib/relativeTime';

type FeedEntry = {
  id: string;
  icon: typeof BuildOutlined;
  message: string;
  createdAt: string;
  onClick: () => void;
};

const PAGE_SIZE = 8;

/**
 * Real, but deliberately scoped: no "Payment received" entries, since
 * there's no payments feature to generate them from (see KpiStrip's
 * identical caveat) — only what's actually trackable. Merges three
 * real sources client-side: portfolio-wide work order activity (its
 * own backend endpoint, since MaintenanceActivity was previously only
 * queryable per-work-order), recently-updated leases, and
 * recently-added vendors.
 */
export function ActivityFeed() {
  const navigate = useNavigate();
  const [visibleCount, setVisibleCount] = useState(PAGE_SIZE);

  const { data: workOrderActivity, isLoading: activityLoading } = useRecentWorkOrderActivity(20, 0);
  const { data: leasesData, isLoading: leasesLoading } = useLeasesPortfolio({ sort: 'startDate', order: 'desc', limit: 10, offset: 0 });
  const { data: vendorsData, isLoading: vendorsLoading } = useVendors({ sort: 'name', order: 'asc', limit: 10, offset: 0 });

  const isLoading = activityLoading || leasesLoading || vendorsLoading;

  const entries: FeedEntry[] = [
    ...(workOrderActivity ?? []).map((a): FeedEntry => ({
      id: `wo-${a.id}`,
      icon: BuildOutlined,
      message: `${a.workOrderTitle} — ${a.message} (${a.propertyName})`,
      createdAt: a.createdAt,
      onClick: () => navigate(`/maintenance`),
    })),
    ...(leasesData?.leases ?? []).map((l): FeedEntry => ({
      id: `lease-${l.id}`,
      icon: DescriptionOutlined,
      // LeaseRow (the list projection) has no createdAt — startDate is
      // the closest available timestamp, hence "starts" rather than a
      // false "created"/"signed" claim about when the record was made.
      message: `Lease ${l.signed ? '(signed)' : '(unsigned)'} starts — ${l.primaryResidentName}, ${l.propertyName} / ${l.unitName}`,
      createdAt: l.startDate,
      onClick: () => navigate(`/leases/${l.id}`),
    })),
    ...(vendorsData?.vendors ?? []).map((v): FeedEntry => ({
      id: `vendor-${v.id}`,
      icon: StorefrontOutlined,
      message: `Vendor added — ${v.companyName}`,
      createdAt: v.createdAt,
      onClick: () => navigate(`/vendors/${v.id}`),
    })),
  ].sort((a, b) => b.createdAt.localeCompare(a.createdAt));

  const visible = entries.slice(0, visibleCount);

  return (
    <Paper variant="outlined" sx={{ p: 2.5, bgcolor: tokens.slate[50] }}>
      <Typography sx={{ fontSize: 12.5, fontWeight: 600, color: tokens.slate[600], mb: 1.5 }}>Recent Activity</Typography>

      {isLoading ? (
        <LoadingSpinner label="Loading…" />
      ) : entries.length === 0 ? (
        <EmptyState icon={BuildOutlined} title="No recent activity" description="Activity across leases, work orders, and vendors will show up here." />
      ) : (
        <Stack spacing={0}>
          {visible.map((e) => {
            const Icon = e.icon;
            return (
              <Stack
                key={e.id}
                direction="row"
                spacing={1.25}
                onClick={e.onClick}
                sx={{
                  alignItems: 'flex-start',
                  py: 0.75,
                  cursor: 'pointer',
                  '&:hover': { '& .feed-message': { color: tokens.slate[900] } },
                }}
              >
                <Icon sx={{ fontSize: 15, color: tokens.slate[400], mt: 0.25 }} />
                <Box sx={{ flex: 1, minWidth: 0 }}>
                  <Typography
                    className="feed-message"
                    sx={{ fontSize: 12.5, color: tokens.slate[600], overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
                  >
                    {e.message}
                  </Typography>
                </Box>
                <Typography sx={{ fontSize: 11.5, color: tokens.slate[400], whiteSpace: 'nowrap' }}>{relativeTime(e.createdAt)}</Typography>
              </Stack>
            );
          })}
          {visibleCount < entries.length && (
            <Button size="small" onClick={() => setVisibleCount((n) => n + PAGE_SIZE)} sx={{ alignSelf: 'flex-start', mt: 1, fontSize: 12 }}>
              Load more
            </Button>
          )}
        </Stack>
      )}
    </Paper>
  );
}
