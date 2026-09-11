import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
// latin-only subsets: this UI is English-only, so the default imports
// (which pull cyrillic/greek/math/etc. subsets too) would ship glyphs
// nothing ever renders.
// Manrope: headings + wordmark.
import '@fontsource/manrope/latin-700.css';
import '@fontsource/manrope/latin-800.css';
// Public Sans: all other UI copy.
import '@fontsource/public-sans/latin-400.css';
import '@fontsource/public-sans/latin-500.css';
import '@fontsource/public-sans/latin-600.css';
// IBM Plex Mono: numeric/monospace labels (metric values, keycap chips).
import '@fontsource/ibm-plex-mono/latin-400.css';
import '@fontsource/ibm-plex-mono/latin-500.css';
import '@fontsource/ibm-plex-mono/latin-600.css';
import { App } from './app/App';
import './index.css';

const rootElement = document.getElementById('root');
if (!rootElement) {
  throw new Error('Root element #root not found');
}

createRoot(rootElement).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
