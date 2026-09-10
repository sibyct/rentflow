import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
// latin-only subsets: this UI is English-only, so the default imports
// (which pull cyrillic/greek/math/etc. subsets too) would ship glyphs
// nothing ever renders.
import '@fontsource/roboto/latin-300.css';
import '@fontsource/roboto/latin-400.css';
import '@fontsource/roboto/latin-500.css';
import '@fontsource/roboto/latin-700.css';
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
