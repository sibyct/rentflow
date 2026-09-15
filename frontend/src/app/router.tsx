import { Navigate, Route, Routes } from 'react-router-dom';
import { AppShell } from './layout/AppShell';
import { AuthLayout } from './layout/AuthLayout';
// ProtectedRoute import intentionally left out while the guard below is
// disabled — see the commented-out <ProtectedRoute> wrapping AppShell.
// Restore this import if/when that's re-enabled.
import { LoginPage } from './pages/LoginPage';
import { RegisterPage } from './pages/RegisterPage';
import { ForgotPasswordPage } from './pages/ForgotPasswordPage';
import { DashboardPage } from './pages/DashboardPage';
import { PropertiesPage } from './pages/PropertiesPage';
import { PropertyDetailPage } from './pages/PropertyDetailPage';
import { NotFoundPage } from './pages/NotFoundPage';
import { PlaceholderPage } from '@/shared/components';

export function AppRouter() {
  return (
    <Routes>
      {/* Unauthenticated pages get their own full-bleed layout — no app nav. */}
      <Route element={<AuthLayout />}>
        <Route path="login" element={<LoginPage />} />
        <Route path="register" element={<RegisterPage />} />
        <Route path="forgot-password" element={<ForgotPasswordPage />} />
      </Route>

      {/* Authenticated app: one ProtectedRoute guarding the whole shell,
          rather than repeating it on every route. */}
      <Route
        element={
          // <ProtectedRoute>
            <AppShell />
          // </ProtectedRoute>
        }
      >
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<DashboardPage />} />

        <Route path="properties" element={<PropertiesPage />} />
        <Route path="properties/:id" element={<PropertyDetailPage />} />

        {/* Nav destinations with no real feature yet — see navConfig.tsx
            and README for how to replace one of these with a real page. */}
        <Route path="units" element={<PlaceholderPage label="Units" />} />
        <Route path="leases" element={<PlaceholderPage label="Leases" />} />
        <Route path="residents" element={<PlaceholderPage label="Residents" />} />
        <Route path="maintenance" element={<PlaceholderPage label="Maintenance" />} />
        <Route path="vendors" element={<PlaceholderPage label="Vendors" />} />
        <Route path="settings" element={<PlaceholderPage label="Settings" />} />

        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
