import { NavLink, Link, Outlet } from 'react-router-dom';
import { Suspense } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  LayoutDashboard,
  PieChart,
  LineChart,
  Settings,
  Menu,
  Wallet,
  Landmark,
  TrendingUp,
} from 'lucide-react';
import { useAuthStore } from '@/stores/authStore';
import { userApi } from '@/api/user';
import { portfolioApi } from '@/api/portfolio';
import { LogoutButton } from '@/components/auth/LogoutButton';
import { Sheet, SheetContent, SheetTrigger } from '@/components/ui/sheet';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { ThemeToggle } from '@/components/ui/ThemeToggle';
import { Skeleton } from '@/components/ui/skeleton';

const navItems = [
  { name: 'Dashboard', path: '/dashboard', icon: LayoutDashboard },
  { name: 'Portfolio', path: '/portfolio', icon: PieChart },
  { name: 'Forecast', path: '/forecast', icon: LineChart },
  { name: 'Settings', path: '/settings', icon: Settings },
];

export function AppShell() {
  const { user, setUser } = useAuthStore();

  // Live wallet query
  useQuery({
    queryKey: ['userBalance'],
    queryFn: async () => {
      const res = await userApi.getUser();
      setUser(res.data);
      return res.data;
    },
    refetchInterval: 10000,
  });

  const { data: portfolioData } = useQuery({
    queryKey: ['portfolio-allocation'],
    queryFn: () => portfolioApi.getPortfolio().then((res) => res.data),
    staleTime: 60_000,
  });

  const portfolioValue = portfolioData?.live_total_value ?? 0;
  const hasPortfolio = portfolioData?.holdings && portfolioData.holdings.length > 0;

  const formatUSD = (v: number) =>
    new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(v);

  const NavLinks = () => (
    <nav className="flex flex-col gap-2">
      {navItems.map((item) => (
        <NavLink
          key={item.path}
          to={item.path}
          className={({ isActive }) =>
            `flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
              isActive
                ? 'bg-primary/10 text-primary'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground'
            }`
          }
        >
          <item.icon className="h-4 w-4" />
          {item.name}
        </NavLink>
      ))}
    </nav>
  );

  return (
    <div className="flex min-h-screen w-full bg-background">
      {/* Desktop Sidebar */}
      <aside className="hidden border-r bg-card md:flex md:w-64 md:flex-col">
        <div className="flex h-16 items-center px-6 border-b">
          <Link to="/dashboard" className="flex items-center gap-2 hover:opacity-80 transition-opacity">
            <Landmark className="h-5 w-5 text-primary" />
            <span className="text-base font-semibold tracking-tight" style={{ fontFamily: "'DM Serif Display', Georgia, serif" }}>RoboAdvisor</span>
          </Link>
        </div>
        <div className="flex-1 overflow-auto py-6 px-4">
          <NavLinks />
        </div>
      </aside>

      {/* Main Content Area */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Header */}
        <header className="flex h-16 items-center justify-between border-b bg-card px-4 md:px-6">
          <div className="flex items-center gap-4">
            {/* Mobile Sidebar Trigger */}
            <Sheet>
              <SheetTrigger asChild>
                <Button variant="ghost" size="icon" className="md:hidden">
                  <Menu className="h-5 w-5" />
                  <span className="sr-only">Toggle navigation menu</span>
                </Button>
              </SheetTrigger>
              <SheetContent side="left" className="w-64 p-0">
                <div className="flex h-16 items-center px-6 border-b">
                  <Landmark className="h-5 w-5 text-primary mr-2" />
                  <span className="text-base font-semibold tracking-tight" style={{ fontFamily: "'DM Serif Display', Georgia, serif" }}>RoboAdvisor</span>
                </div>
                <div className="py-6 px-4">
                  <NavLinks />
                </div>
              </SheetContent>
            </Sheet>

            {/* Title / Breadcrumb can go here */}
          </div>

          <div className="flex items-center gap-3">
            {/* Wallet Balance Badge */}
            <Badge variant="secondary" className="hidden sm:flex h-9 px-4 py-2 gap-2 text-sm">
              <Wallet className="h-4 w-4 text-primary" />
              <span>{formatUSD(user?.wallet_balance || 0)}</span>
            </Badge>

            {/* Portfolio Value Badge */}
            {hasPortfolio && (
              <Badge variant="outline" className="hidden sm:flex h-9 px-4 py-2 gap-2 text-sm border-emerald-500/40 text-emerald-600 dark:text-emerald-400">
                <TrendingUp className="h-4 w-4" />
                <span>{formatUSD(portfolioValue)}</span>
              </Badge>
            )}

            <ThemeToggle />
            <Separator orientation="vertical" className="h-8 hidden sm:block" />

            {/* User Profile / Logout */}
            <div className="flex items-center gap-3">
              <Link
                to="/settings"
                className="text-sm font-medium hidden sm:block truncate max-w-[220px] hover:text-primary transition-colors"
                title={user?.email}
              >
                {user?.email?.split("@")[0]}
              </Link>
              <LogoutButton variant="outline" className="h-9 px-3" showText={false} />
            </div>
          </div>
        </header>

        {/* Page Content */}
        <main className="flex-1 overflow-auto bg-muted/20">
          <Suspense
            fallback={
              <div className="p-6 space-y-6">
                <Skeleton className="h-[120px] w-full max-w-sm rounded-xl" />
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                  <Skeleton className="h-[300px] rounded-xl" />
                  <Skeleton className="h-[300px] rounded-xl" />
                </div>
              </div>
            }
          >
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  );
}
