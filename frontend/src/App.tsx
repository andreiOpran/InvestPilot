import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';

import { useSilentRestore } from '@/hooks/useSilentRestore';
import { ProtectedRoute } from '@/components/layout/ProtectedRoute';

// Auth pages
import { Register } from '@/pages/auth/Register';
import { RegisterSuccess } from '@/pages/auth/RegisterSuccess';
import { VerifyEmail } from '@/pages/auth/VerifyEmail';
import { Login } from '@/pages/auth/Login';
import { ForgotPassword } from '@/pages/auth/ForgotPassword';
import { ResetPassword } from '@/pages/auth/ResetPassword';
import { LogoutButton } from '@/components/auth/LogoutButton';
import { Settings } from '@/pages/Settings';
import { Onboarding } from '@/pages/Onboarding';

function AppRoutes() {
  // restore session silently on mount
  useSilentRestore();

  return (
    <Routes>
      {/* Public auth routes */}
      <Route path="/register" element={<Register />} />
      <Route path="/register-success" element={<RegisterSuccess />} />
      <Route path="/verify-email" element={<VerifyEmail />} />
      <Route path="/login" element={<Login />} />
      <Route path="/forgot-password" element={<ForgotPassword />} />
      <Route path="/reset-password" element={<ResetPassword />} />

      {/* Protected routes (require authentication) */}
      <Route element={<ProtectedRoute />}>
        <Route path="/dashboard" element={
          <div className="p-8">
            <h1 className="text-2xl font-bold">Dashboard</h1>
            <p className="text-muted-foreground mt-2 mb-6">Coming soon...</p>
            <LogoutButton />
          </div>
        } />
        <Route path="/settings" element={<Settings />} />
        <Route path="/onboarding" element={<Onboarding />} />
      </Route>

      {/* Fallback */}
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  );
}

function App() {
  return (
    <BrowserRouter>
      <AppRoutes />
    </BrowserRouter>
  );
}

export default App;
