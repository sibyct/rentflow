import { useState } from 'react';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { QueryState } from '@/shared/components';
import { useProperties } from '../hooks/usePropertyQueries';
import { PropertyCard } from './PropertyCard';

const PAGE_SIZE = 20;

export function PropertyList() {
  const [offset, setOffset] = useState(0);
  const { data, isLoading, isError, error, refetch } = useProperties({ limit: PAGE_SIZE, offset });

  return (
    <QueryState isLoading={isLoading} isError={isError} error={error} onRetry={() => void refetch()}>
      {data && data.properties.length === 0 && (
        <Typography variant="body2" color="text.secondary" sx={{ py: 4, textAlign: 'center' }}>
          No properties yet.
        </Typography>
      )}

      {data && data.properties.length > 0 && (
        <>
          <Box
            sx={{
              display: 'grid',
              gap: 2,
              gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)', lg: 'repeat(3, 1fr)' },
            }}
          >
            {data.properties.map((property) => (
              <PropertyCard key={property.id} property={property} />
            ))}
          </Box>

          <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mt: 3 }}>
            <Typography variant="body2" color="text.secondary">
              Showing {offset + 1}–{Math.min(offset + PAGE_SIZE, data.meta.total)} of {data.meta.total}
            </Typography>
            <Stack direction="row" spacing={1}>
              <Button
                variant="outlined"
                size="small"
                onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
                disabled={offset === 0}
              >
                Previous
              </Button>
              <Button
                variant="outlined"
                size="small"
                onClick={() => setOffset((o) => o + PAGE_SIZE)}
                disabled={offset + PAGE_SIZE >= data.meta.total}
              >
                Next
              </Button>
            </Stack>
          </Stack>
        </>
      )}
    </QueryState>
  );
}
