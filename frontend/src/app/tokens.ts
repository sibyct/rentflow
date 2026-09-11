// Design tokens that don't fit MUI's standard Palette shape (extra azure/
// slate steps, fixed shadows, the shell's own layout constants). Theme
// palette/typography live in theme.ts; anything referenced by the app
// shell specifically that isn't a first-class MUI palette slot lives
// here instead of as one-off hex values scattered through components.
export const tokens = {
  azure: {
    50: '#EEF3FE',
    200: '#B2C8FA',
    500: '#1A56DB',
    600: '#1546B4',
    700: '#113791',
  },
  slate: {
    50: '#F7F8FA',
    100: '#EDEFF3',
    200: '#DDE1E8',
    300: '#C3C9D4',
    400: '#98A1B2',
    500: '#6B7688',
    600: '#4E5868',
    700: '#39414E',
    900: '#14181F',
  },
  error: '#C2262F',
  warningTint: '#FDF4E3',
  warningInk: '#7A4800',
  successInk: '#06403A',

  radiusControl: 6,
  radiusCard: 12,
  shadowCard: '0 1px 2px rgba(20,24,31,0.08)',
  shadowPopover: '0 4px 12px rgba(20,24,31,0.10)',
  focusRing: '0 0 0 3px rgba(26,86,219,0.32)',

  fontMono: '"IBM Plex Mono", "Courier New", monospace',

  navWidthExpanded: 248,
  navWidthCollapsed: 64,
  topBarHeight: 56,
  navItemHeight: 40,
  collapseBreakpoint: 1100,
} as const;
