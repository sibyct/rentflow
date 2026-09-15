import { useState, type MouseEvent } from 'react';
import { useLocation } from 'react-router-dom';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import KeyboardArrowDownOutlined from '@mui/icons-material/KeyboardArrowDownOutlined';
import { tokens } from '../tokens';
import { findNavItemByPath } from './navConfig';

const RANGE_OPTIONS = ['This month', 'Last month', 'This quarter', 'This year'];

// Detail-style pages (one record, not a list) build a richer breadcrumb
// + title + actions row of their own — see PropertyDetailScreen — which
// makes this generic one directly redundant, not just visually similar.
// Add a pattern here for any future page in the same situation (e.g. a
// unit or lease detail page) rather than special-casing AppShell/routing.
const OWN_HEADER_ROUTE_PATTERNS = [/^\/properties\/[^/]+$/];

/**
 * Breadcrumb + H1 are entirely data-driven from navConfig — they update
 * automatically as the route changes, so individual pages never need to
 * set their own header. The "This month" range dropdown + Export button
 * are Dashboard-specific (revenue range), not a generic per-page toolbar,
 * so they only render there — every other page's own content owns
 * whatever actions/subtext belong under its H1 (see PropertiesScreen for
 * an example with a real, working Export button of its own).
 */
export function PageHeader() {
  const location = useLocation();
  const match = findNavItemByPath(location.pathname);
  const [range, setRange] = useState(RANGE_OPTIONS[0]);
  const [menuAnchor, setMenuAnchor] = useState<HTMLElement | null>(null);

  if (!match) return null;
  if (OWN_HEADER_ROUTE_PATTERNS.some((pattern) => pattern.test(location.pathname))) return null;

  const handleOpenMenu = (e: MouseEvent<HTMLElement>) => setMenuAnchor(e.currentTarget);
  const handleSelect = (option: string) => {
    setRange(option);
    setMenuAnchor(null);
  };

  return (
    <Stack direction="row" sx={{ alignItems: 'flex-start', justifyContent: 'space-between', mb: 3 }}>
      <Box>
        <Typography sx={{ fontSize: 13, color: tokens.slate[500], mb: 0.5 }}>{match.groupLabel}</Typography>
        <Typography variant="h3">{match.item.label}</Typography>
      </Box>

      {match.item.id === 'dashboard' && (
        <Stack direction="row" spacing={1.5}>
          <Button
            variant="outlined"
            onClick={handleOpenMenu}
            endIcon={<KeyboardArrowDownOutlined />}
            sx={{ borderColor: tokens.slate[300], color: tokens.slate[700] }}
          >
            {range}
          </Button>
          <Menu anchorEl={menuAnchor} open={Boolean(menuAnchor)} onClose={() => setMenuAnchor(null)}>
            {RANGE_OPTIONS.map((option) => (
              <MenuItem key={option} selected={option === range} onClick={() => handleSelect(option)}>
                {option}
              </MenuItem>
            ))}
          </Menu>

          <Button variant="outlined" disabled sx={{ borderColor: tokens.slate[300] }}>
            Export
          </Button>
        </Stack>
      )}
    </Stack>
  );
}
