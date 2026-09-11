import type { ComponentType } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Typography from '@mui/material/Typography';
import type { SvgIconProps } from '@mui/material/SvgIcon';
import { tokens } from '@/app/tokens';
import { useDimChromeWhileMounted } from '@/app/layout/ChromeInteractivityContext';

export interface ErrorStateAction {
  label: string;
  /** Client-side navigation (e.g. "Go to dashboard"). Mutually exclusive with onClick in practice, but both are wired if given. */
  href?: string;
  onClick?: () => void;
}

export interface ErrorStateProps {
  /** A simple line-art MUI icon — kept muted via `tone`, never rendered raw/red. */
  icon: ComponentType<SvgIconProps>;
  title: string;
  message: string;
  /** The one action that actually resolves this error — omit rather than fall back to a mismatched "Try again". */
  primaryAction?: ErrorStateAction;
  secondaryAction?: ErrorStateAction;
  /** 'destructive' is for account/data-loss-adjacent failures; every other error stays in the calm neutral palette. */
  tone?: 'neutral' | 'destructive';
  /**
   * Fades and disables the sidebar/top bar's session-dependent controls
   * (search, notifications, account menu, nav links) for as long as this
   * error is on screen — appropriate for session-expiry and offline states,
   * where those controls can't actually do anything right now. Leave off
   * for scoped failures (404, 403, a single failed request) where the rest
   * of the app is still perfectly usable.
   */
  dimChrome?: boolean;
}

/**
 * Generic full-bleed error/empty state for the app shell's content area.
 * Renders in place of a page's normal content — the sidebar and top bar
 * stay mounted around it (see AppShell), so the user keeps their bearings
 * instead of facing a blank screen. Configure per error type via props
 * rather than branching internally; see ERROR_STATE_PRESETS below for the
 * standard 401/403/404/500/offline configs.
 */
export function ErrorState({ icon: Icon, title, message, primaryAction, secondaryAction, tone = 'neutral', dimChrome = false }: ErrorStateProps) {
  useDimChromeWhileMounted(dimChrome);

  const palette = tone === 'destructive' ? { bg: '#FDECEC', fg: tokens.error } : { bg: tokens.azure[50], fg: tokens.azure[600] };

  return (
    <Box
      sx={{
        minHeight: 'clamp(360px, 60vh, 640px)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        textAlign: 'center',
        px: 3,
        py: 6,
      }}
    >
      <Box sx={{ maxWidth: 480, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
        <Box
          sx={{
            width: 96,
            height: 96,
            flexShrink: 0,
            borderRadius: '50%',
            bgcolor: palette.bg,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            mb: 3,
          }}
        >
          <Icon sx={{ fontSize: 44, color: palette.fg }} />
        </Box>

        <Typography sx={{ fontSize: { xs: 20, sm: 24 }, fontWeight: 700, color: 'text.primary', mb: 1 }}>{title}</Typography>

        <Typography sx={{ fontSize: 15, lineHeight: 1.6, color: tokens.slate[500], mb: 4 }}>{message}</Typography>

        {primaryAction && (
          <Button
            variant="contained"
            component={primaryAction.href ? RouterLink : 'button'}
            to={primaryAction.href}
            onClick={primaryAction.onClick}
            sx={{ minWidth: 180 }}
          >
            {primaryAction.label}
          </Button>
        )}

        {secondaryAction && (
          <Button
            variant="text"
            component={secondaryAction.href ? RouterLink : 'button'}
            to={secondaryAction.href}
            onClick={secondaryAction.onClick}
            sx={{
              mt: 1.5,
              fontSize: 13,
              fontWeight: 600,
              color: tokens.slate[500],
              '&:hover': { bgcolor: 'transparent', color: tokens.slate[700], textDecoration: 'underline' },
            }}
          >
            {secondaryAction.label}
          </Button>
        )}
      </Box>
    </Box>
  );
}
