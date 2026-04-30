import { Link } from 'react-router-dom';
import { Landmark, TrendingUp, ShieldCheck, BarChart3 } from 'lucide-react';
import { Button } from '@/components/ui/button';

export function Landing() {
  return (
    <div className="flex min-h-screen flex-col bg-background">
      {/* Nav */}
      <header className="flex h-16 items-center justify-between px-6 border-b">
        <div className="flex items-center gap-2">
          <Landmark className="h-6 w-6 text-primary" />
          <span className="text-lg font-bold">RoboAdvisor</span>
        </div>
        <div className="flex items-center gap-3">
          <Button variant="ghost" asChild>
            <Link to="/login">Log in</Link>
          </Button>
          <Button asChild>
            <Link to="/register">Get started</Link>
          </Button>
        </div>
      </header>

      {/* Hero */}
      <main className="flex flex-1 flex-col items-center justify-center text-center px-6 gap-8">
        <div className="space-y-4 max-w-2xl">
          <h1 className="text-5xl font-extrabold tracking-tight">
            Automated investing,{' '}
            <span className="text-primary">intelligently managed.</span>
          </h1>
          <p className="text-lg text-muted-foreground">
            Set your risk profile once. RoboAdvisor builds and rebalances your portfolio using Hierarchical Risk Parity — hands-free.
          </p>
        </div>

        <div className="flex gap-4 flex-wrap justify-center">
          <Button size="lg" asChild>
            <Link to="/register">Start for free</Link>
          </Button>
          <Button size="lg" variant="outline" asChild>
            <Link to="/login">Log in</Link>
          </Button>
        </div>

        {/* Feature pills */}
        <div className="flex flex-wrap justify-center gap-6 text-sm text-muted-foreground mt-4">
          <div className="flex items-center gap-2">
            <TrendingUp className="h-4 w-4 text-primary" />
            HRP portfolio optimization
          </div>
          <div className="flex items-center gap-2">
            <BarChart3 className="h-4 w-4 text-primary" />
            Monte Carlo forecasting
          </div>
          <div className="flex items-center gap-2">
            <ShieldCheck className="h-4 w-4 text-primary" />
            2FA &amp; bank-grade security
          </div>
        </div>
      </main>
    </div>
  );
}
