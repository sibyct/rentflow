import { Navigate, Route, Routes } from 'react-router-dom';
import { AppShell } from './layout/AppShell';
import { AuthLayout } from './layout/AuthLayout';
import { ProtectedRoute } from './layout/ProtectedRoute';
import { LoginPage } from './pages/LoginPage';
import { RegisterPage } from './pages/RegisterPage';
import { ForgotPasswordPage } from './pages/ForgotPasswordPage';
import { AcceptInvitePage } from './pages/AcceptInvitePage';
import { ResetPasswordPage } from './pages/ResetPasswordPage';
import { DashboardPage } from './pages/DashboardPage';
import { PropertiesPage } from './pages/PropertiesPage';
import { PropertyDetailPage } from './pages/PropertyDetailPage';
import { UnitsPage } from './pages/UnitsPage';
import { UnitDetailPage } from './pages/UnitDetailPage';
import { LeasesPage } from './pages/LeasesPage';
import { LeaseDetailPage } from './pages/LeaseDetailPage';
import { MaintenancePage } from './pages/MaintenancePage';
import { VendorsPage } from './pages/VendorsPage';
import { VendorDetailPage } from './pages/VendorDetailPage';
import { AccountingDashboardPage } from './pages/AccountingDashboardPage';
import { RentRollPage } from './pages/RentRollPage';
import { ExpensesPage } from './pages/ExpensesPage';
import { ChargesPage } from './pages/ChargesPage';
import { AccountingLayout } from '@/features/accounting';
import { SettingsPage } from './pages/SettingsPage';
import { UsersRolesPage } from './pages/UsersRolesPage';
import { NotFoundPage } from './pages/NotFoundPage';

export function AppRouter() {
  return (
    <Routes>
      {/* Unauthenticated pages get their own full-bleed layout — no app nav. */}
      <Route element={<AuthLayout />}>
        <Route path="login" element={<LoginPage />} />
        <Route path="register" element={<RegisterPage />} />
        <Route path="forgot-password" element={<ForgotPasswordPage />} />
        {/* Where a staff invite email links to (?token=...) — pre-auth
            like the routes above it, so it shares this same split layout
            rather than the app shell. */}
        <Route path="accept-invite" element={<AcceptInvitePage />} />
        {/* Where an admin-triggered "reset password" email links to
            (?token=...) — same pre-auth split layout as above. */}
        <Route path="reset-password" element={<ResetPasswordPage />} />
      </Route>

      {/* Authenticated app: one ProtectedRoute guarding the whole shell,
          rather than repeating it on every route. Also what redirects
          to /login the moment a session expires mid-use — client.ts's
          requestEnvelope calls useAuthStore.clear() when a 401's
          refresh attempt fails, flipping isAuthenticated to false,
          which this re-renders on and redirects from immediately. */}
      <Route
        element={
          <ProtectedRoute>
            <AppShell />
          </ProtectedRoute>
        }
      >
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<DashboardPage />} />

        <Route path="properties" element={<PropertiesPage />} />
        <Route path="properties/:id" element={<PropertyDetailPage />} />

        <Route path="units" element={<UnitsPage />} />
        <Route path="units/:id" element={<UnitDetailPage />} />

        <Route path="leases" element={<LeasesPage />} />
        <Route path="leases/:id" element={<LeaseDetailPage />} />

        <Route path="maintenance" element={<MaintenancePage />} />

        <Route path="vendors" element={<VendorsPage />} />
        <Route path="vendors/:id" element={<VendorDetailPage />} />

        {/* Accounting: a shared tabbed layout, one real route per tab. */}
        <Route path="accounting" element={<AccountingLayout />}>
          <Route index element={<AccountingDashboardPage />} />
          <Route path="rent-roll" element={<RentRollPage />} />
          <Route path="expenses" element={<ExpensesPage />} />
          <Route path="charges" element={<ChargesPage />} />
        </Route>

        <Route path="settings" element={<SettingsPage />} />
        <Route path="settings/users-roles" element={<UsersRolesPage />} />

        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
