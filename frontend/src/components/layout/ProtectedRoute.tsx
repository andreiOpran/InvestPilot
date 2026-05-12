import { Navigate, useLocation } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import { Skeleton } from '@/components/ui/skeleton';
import { AppShell } from '@/components/layout/AppShell';

export function ProtectedRoute() {
  const { status, user } = useAuthStore();
  const location = useLocation();

  // Show the loading skeleton ONLY when we have no user yet.
  if (status === 'loading' && !user) {
    return (
      <div className="flex h-screen w-full bg-background">
        {/* Sidebar Skeleton */}
        <div className="hidden md:flex w-64 flex-col border-r bg-card">
          <div className="flex h-16 items-center px-6 border-b gap-2">
            <Skeleton className="h-5 w-5 rounded" />
            <Skeleton className="h-5 w-24" />
          </div>
          <div className="flex-1 py-6 px-4 space-y-2">
            {[...Array(4)].map((_, i) => (
              <Skeleton key={i} className="h-9 w-full rounded-lg" />
            ))}
          </div>
        </div>
        {/* Main Content Skeleton */}
        <div className="flex flex-1 flex-col overflow-hidden">
          {/* Header Skeleton */}
          <div className="h-16 border-b bg-card flex items-center px-4 md:px-6 justify-end gap-3">
            <Skeleton className="h-9 w-28 rounded-md hidden sm:block" />
            <Skeleton className="h-9 w-9 rounded-md" />
            <Skeleton className="h-8 w-px hidden sm:block" />
            <Skeleton className="h-4 w-20 hidden sm:block" />
            <Skeleton className="h-9 w-9 rounded-md" />
          </div>
          {/* Body Skeleton — matches Dashboard layout */}
          <div className="flex-1 overflow-auto bg-muted/20">
            <div className="p-6 md:p-8 space-y-6 max-w-7xl mx-auto">
              {/* Page header */}
              <div className="space-y-1.5">
                <Skeleton className="h-6 w-32" />
                <Skeleton className="h-4 w-48" />
              </div>
              {/* KPI grid */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <Skeleton className="h-[160px] rounded-xl" />
                <Skeleton className="h-[160px] rounded-xl" />
                <Skeleton className="h-[160px] rounded-xl" />
              </div>
              {/* Charts grid */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Skeleton className="h-[320px] rounded-xl" />
                <Skeleton className="h-[320px] rounded-xl" />
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (status === 'unauthenticated') {
    return <Navigate to="/" state={{ from: location }} replace />;
  }

  // If authenticated but onboarding is not complete
  if (status === 'authenticated' && user && user.risk_tolerance === 0 && user.investment_horizon === 0) {
    const isOnboardingPath = location.pathname === '/onboarding';
    const isSettingsPath = location.pathname === '/settings';
    
    // Check if the user is already on a path they are allowed to be on
    if (!isOnboardingPath && !isSettingsPath) {
      return <Navigate to="/onboarding" replace />;
    }
  }

  return <AppShell />;
}


