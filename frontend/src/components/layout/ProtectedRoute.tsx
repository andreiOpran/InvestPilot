import { Navigate, Outlet } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import { Skeleton } from '@/components/ui/skeleton';

export function ProtectedRoute() {
  const { status, user } = useAuthStore();

  // Show the loading skeleton ONLY when we have no user yet.
  // If we already have a user (e.g. right after a fresh login while useSilentRestore
  // briefly flips status back to 'loading'), render the outlet to avoid a blank screen.
  if (status === 'loading' && !user) {
    return (
      <div className="flex h-screen w-full bg-background">
        {/* Sidebar Skeleton */}
        <div className="hidden md:flex w-64 flex-col border-r p-6 gap-6">
          <Skeleton className="h-8 w-32 mb-6" />
          <div className="space-y-4">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-5/6" />
            <Skeleton className="h-4 w-full" />
          </div>
        </div>
        {/* Main Content Skeleton */}
        <div className="flex-1 flex flex-col">
          {/* Header Skeleton */}
          <div className="h-16 border-b flex items-center px-6 justify-end gap-4">
            <Skeleton className="h-8 w-24 rounded-full" />
            <Skeleton className="h-8 w-8 rounded-full" />
          </div>
          {/* Body Skeleton */}
          <div className="p-6 space-y-6">
            <Skeleton className="h-[120px] w-full max-w-sm rounded-xl" />
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <Skeleton className="h-[300px] rounded-xl" />
              <Skeleton className="h-[300px] rounded-xl" />
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (status === 'unauthenticated') {
    return <Navigate to="/login" replace />;
  }

  if (status === 'authenticated' && user && user.risk_tolerance === 0) {
    // Only redirect to onboarding if they are not already going there or to settings
    const isAllowedPath = window.location.pathname === '/onboarding' || window.location.pathname === '/settings';
    if (!isAllowedPath) {
      return <Navigate to="/onboarding" replace />;
    }
  }

  return <Outlet />;
}

