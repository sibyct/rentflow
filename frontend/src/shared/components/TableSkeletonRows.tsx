import Skeleton from '@mui/material/Skeleton';
import Stack from '@mui/material/Stack';
import TableCell from '@mui/material/TableCell';
import TableRow from '@mui/material/TableRow';

interface SkeletonShape {
  variant?: 'text' | 'rounded' | 'circular';
  width?: number | string;
  height?: number | string;
}

export interface TableSkeletonColumn {
  align?: 'left' | 'right' | 'center';
  padding?: 'checkbox' | 'none' | 'normal';
  /** One shape per line in the cell — two stacked shapes reads as a title + subtitle. */
  shapes: SkeletonShape[];
}

interface TableSkeletonRowsProps {
  /** Number of placeholder rows to render while real data is loading. Default 5. */
  rows?: number;
  columns: TableSkeletonColumn[];
}

/**
 * Placeholder `<TableRow>`s for a loading table body — drop it inside a
 * `<TableBody>` alongside (never instead of) the real rows, gated on
 * your own loading flag. `columns` describes one table's worth of
 * skeleton shapes once; reuse the same array across renders rather than
 * re-declaring it inline, since it doesn't depend on any row data.
 */
export function TableSkeletonRows({ rows = 5, columns }: TableSkeletonRowsProps) {
  return (
    <>
      {Array.from({ length: rows }).map((_, rowIndex) => (
        <TableRow key={rowIndex}>
          {columns.map((col, colIndex) => (
            <TableCell key={colIndex} align={col.align} padding={col.padding}>
              {col.shapes.length > 0 && (
                <Stack spacing={0.5} sx={col.align === 'right' ? { alignItems: 'flex-end' } : undefined}>
                  {col.shapes.map((shape, shapeIndex) => (
                    <Skeleton key={shapeIndex} variant={shape.variant ?? 'text'} width={shape.width} height={shape.height} />
                  ))}
                </Stack>
              )}
            </TableCell>
          ))}
        </TableRow>
      ))}
    </>
  );
}
