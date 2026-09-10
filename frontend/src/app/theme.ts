import { createTheme } from '@mui/material/styles';

// Central place to customize MUI's design tokens (palette, typography,
// component defaults). Import `theme` wherever a component needs to
// read a token directly; otherwise components pick it up automatically
// through the ThemeProvider in app/providers.tsx.
export const theme = createTheme({
  palette: {
    mode: 'light',
    primary:   { main: '#1A56DB', light: '#4C7BEC', dark: '#1546B4', contrastText: '#FFFFFF' },
    secondary: { main: '#0F8577', light: '#34A093', dark: '#0B6A5F', contrastText: '#FFFFFF' },
    error:     { main: '#C2262F', light: '#E8737C', dark: '#9B1C24' },
    warning:   { main: '#B26A00', light: '#E8A33D', dark: '#7A4800' },
    success:   { main: '#0F8577', light: '#34A093', dark: '#06403A' },
    info:      { main: '#0B6EA8', light: '#3E9BD1', dark: '#08517C' },
    grey:      { 50:'#F7F8FA',100:'#EDEFF3',200:'#DDE1E8',300:'#C3C9D4',400:'#98A1B2',
                 500:'#6B7688',600:'#4E5868',700:'#39414E',800:'#262C36',900:'#14181F' },
    background: { default: '#F7F8FA', paper: '#FFFFFF' },
    // primary is grey.800, not grey.900: near-black (#14181F) measures
    // ~17:1 against white — far past AAA's 7:1 minimum, and past that
    // point extra contrast just reads as harsh in dense body copy.
    // grey.800 still measures ~13.5:1, comfortably AAA, but softer.
    text: { primary: '#262C36', secondary: '#4E5868', disabled: '#98A1B2' },
    divider: '#DDE1E8',
  },
  shape: { borderRadius: 6 },
  spacing: 8,
  breakpoints: { values: { xs: 0, sm: 600, md: 900, lg: 1200, xl: 1536 } },
  typography: {
    fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
    h1: { fontWeight: 700, fontSize: '3rem',    lineHeight: '3.5rem', letterSpacing: '-0.02em' },
    h2: { fontWeight: 700, fontSize: '1.875rem', lineHeight: '2.375rem' },
    h3: { fontWeight: 700, fontSize: '1.5rem',   lineHeight: '2rem' },
    h4: { fontWeight: 700, fontSize: '1.25rem',  lineHeight: '1.75rem' },
    subtitle1: { fontWeight: 600, fontSize: '1rem',    lineHeight: '1.5rem' },
    body1:     { fontWeight: 400, fontSize: '1rem',    lineHeight: '1.625rem' },
    body2:     { fontWeight: 400, fontSize: '0.875rem', lineHeight: '1.375rem' },
    button:    { fontWeight: 600, fontSize: '0.875rem', textTransform: 'none' },
    caption:   { fontWeight: 400, fontSize: '0.75rem',  lineHeight: '1rem' },
    overline:  { fontWeight: 600, fontSize: '0.6875rem', letterSpacing: '0.14em' },
  },
  components: {
    MuiButton:    { defaultProps: { disableElevation: true }, styleOverrides: { root: { height: 40, paddingInline: 16 } } },
    MuiTextField: { defaultProps: { variant: 'outlined', size: 'small', slotProps: { inputLabel: { shrink: true } } } },
    MuiPaper:     { styleOverrides: { root: { borderRadius: 12, border: '1px solid #DDE1E8' } } },
    MuiTableCell: { styleOverrides: { root: { height: 56, borderColor: '#EDEFF3' } } },
    // Browsers apply their own faint placeholder opacity by default
    // (often well under WCAG AA's 4.5:1 against a white field). Pin it
    // to text.secondary at full opacity, which measures ~7.3:1 on white.
    MuiInputBase: {
      styleOverrides: {
        input: {
          '&::placeholder': { color: '#4E5868', opacity: 1 },
        },
      },
    },
  },
})
