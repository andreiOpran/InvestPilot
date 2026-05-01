import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { lazy } from 'react';

import { useSilentRestore } from '@/hooks/useSilentRestore';
import { ProtectedRoute } from '@/components/layout/ProtectedRoute';

import { Landing } from '@/pages/Landing';

// Auth pages (eager - needed immediately)
import { Register } from '@/pages/auth/Register';
import { RegisterSuccess } from '@/pages/auth/RegisterSuccess';
import { VerifyEmail } from '@/pages/auth/VerifyEmail';
import { Login } from '@/pages/auth/Login';
import { ForgotPassword } from '@/pages/auth/ForgotPassword';
import { ResetPassword } from '@/pages/auth/ResetPassword';

// Protected pages (lazy - loaded on demand)
const Settings = lazy(() => import('@/pages/Settings').then(m => ({ default: m.Settings })));
const Onboarding = lazy(() => import('@/pages/Onboarding').then(m => ({ default: m.Onboarding })));
const Dashboard = lazy(() => import('@/pages/Dashboard').then(m => ({ default: m.Dashboard })));
const Portfolio = lazy(() => import('@/pages/Portfolio').then(m => ({ default: m.Portfolio })));
const Forecast = lazy(() => import('@/pages/Forecast').then(m => ({ default: m.Forecast })));

function AppRoutes() {
  // restore session silently on mount
  useSilentRestore();

  return (
    <Routes>
      {/* Landing */}
      <Route path="/" element={<Landing />} />

      {/* Public auth routes */}
      <Route path="/register" element={<Register />} />
      <Route path="/register-success" element={<RegisterSuccess />} />
      <Route path="/verify-email" element={<VerifyEmail />} />
      <Route path="/login" element={<Login />} />
      <Route path="/forgot-password" element={<ForgotPassword />} />
      <Route path="/reset-password" element={<ResetPassword />} />

      {/* Protected routes (require authentication) */}
      <Route element={<ProtectedRoute />}>
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/portfolio" element={<Portfolio />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="/onboarding" element={<Onboarding />} />
        <Route path="/forecast" element={<Forecast />} />
      </Route>

      {/* Fallback */}
      <Route path="*" element={<Navigate to="/" replace />} />
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
