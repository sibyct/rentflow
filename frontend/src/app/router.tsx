import { Navigate, Route, Routes } from 'react-router-dom';
import { RootLayout } from './layout/RootLayout';
import { AuthLayout } from './layout/AuthLayout';
import { ProtectedRoute } from './layout/ProtectedRoute';
import { LoginPage } from './pages/LoginPage';
import { RegisterPage } from './pages/RegisterPage';
import { ForgotPasswordPage } from './pages/ForgotPasswordPage';
import { PropertiesPage } from './pages/PropertiesPage';
import { NewPropertyPage } from './pages/NewPropertyPage';
import { PropertyDetailPage } from './pages/PropertyDetailPage';
import { NotFoundPage } from './pages/NotFoundPage';

export function AppRouter() {
  return (
    <Routes>
      {/* Unauthenticated pages get their own full-bleed layout — no app nav. */}
      <Route element={<AuthLayout />}>
        <Route path="login" element={<LoginPage />} />
        <Route path="register" element={<RegisterPage />} />
        <Route path="forgot-password" element={<ForgotPasswordPage />} />
      </Route>

      <Route element={<RootLayout />}>
        <Route index element={<Navigate to="/properties" replace />} />

        <Route
          path="properties"
          element={
            <ProtectedRoute>
              <PropertiesPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="properties/new"
          element={
            <ProtectedRoute>
              <NewPropertyPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="properties/:id"
          element={
            <ProtectedRoute>
              <PropertyDetailPage />
            </ProtectedRoute>
          }
        />

        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
