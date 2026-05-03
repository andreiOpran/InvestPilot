import { useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import {
  Navigation,
  TrendingUp,
  BarChart3,
  ShieldCheck,
  ArrowRight,
  RefreshCw,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { ThemeToggle } from '@/components/ui/ThemeToggle';

const displayFont = { fontFamily: "'DM Serif Display', Georgia, serif" };

const stats = [
  { value: '10,000+', label: 'Monte Carlo simulations per forecast', index: '01' },
  { value: '19', label: 'ETF asset universe', index: '02' },
  { value: 'Monthly', label: 'Automated rebalancing', index: '03' },
  { value: '5', label: 'Configurable risk profiles', index: '04' },
];

const features = [
  {
    icon: TrendingUp,
    title: 'Hierarchical Risk Parity',
    description:
      'HRP allocates capital based on risk contribution rather than arbitrary weights. Your portfolio is mathematically balanced across equities and bonds every month.',
    index: '01',
  },
  {
    icon: BarChart3,
    title: 'Monte Carlo Forecasting',
    description:
      'Run 10,000 simulated portfolio scenarios using Geometric Brownian Motion. Visualise the cone of uncertainty across your chosen investment horizon.',
    index: '02',
  },
  {
    icon: ShieldCheck,
    title: 'Institutional Security',
    description:
      'TOTP two-factor authentication, refresh token rotation with reuse detection, IP-based rate limiting, and AES-256 encrypted secrets.',
    index: '03',
  },
  {
    icon: RefreshCw,
    title: 'Passive Rebalancing',
    description:
      'Set your risk profile once. The engine evaluates drift thresholds every 30 days and rebalances only when delta exceeds tolerance, minimising unnecessary trades.',
    index: '04',
  },
];

const allocationSegments = [
  { pct: 36, label: 'US Equities', opacity: 1 },         // VTI, VOO, QQQ, IWM, VTV, VUG, Sectors
  { pct: 22, label: 'Intl Equities', opacity: 0.85 },    // VEA, VWO
  { pct: 16, label: 'Fixed Income', opacity: 0.70 },     // BND, TLT, BNDX
  { pct: 12, label: 'Real Assets', opacity: 0.55 },      // VNQ, VNQI, XLE
  { pct: 10, label: 'Corporate Bonds', opacity: 0.40 },  // LQD, HYG
  { pct: 4, label: 'Cash', opacity: 0.25 },              // Liquidity/SGOV
];

const tickerItems = [
  { symbol: 'VTI', change: '+0.42%', positive: true },  // US Equities
  { symbol: 'VOO', change: '-0.01%', positive: false }, // US Equities (Correlated dip)
  { symbol: 'QQQ', change: '+0.18%', positive: true },  // US Equities (Tech lead)
  { symbol: 'VEA', change: '+0.32%', positive: true },  // Intl Equities
  { symbol: 'VWO', change: '-0.25%', positive: false }, // Intl Equities (Emerging lag)
  { symbol: 'VNQ', change: '+0.55%', positive: true },  // Real Assets (REITs)
  { symbol: 'BND', change: '-0.08%', positive: false }, // Fixed Income (Total Bond)
  { symbol: 'HYG', change: '+0.24%', positive: true },  // Corporate Bonds (High Yield)
];

function AllocationBar() {
  return (
    <div className="w-full space-y-3">
      <div className="flex h-1.5 w-full overflow-hidden rounded-full gap-[2px]">
        {allocationSegments.map((s) => (
          <div
            key={s.label}
            style={{
              width: `${s.pct}%`,
              backgroundColor: `var(--primary)`,
              opacity: s.opacity,
            }}
          />
        ))}
      </div>
      <div className="flex flex-wrap gap-x-4 gap-y-1">
        {allocationSegments.map((s) => (
          <div key={s.label} className="flex items-center gap-1.5">
            <div
              className="h-1.5 w-1.5 rounded-full flex-shrink-0"
              style={{ backgroundColor: `var(--primary)`, opacity: s.opacity }}
            />
            <span className="text-[10px] text-muted-foreground/70 tracking-wide">{s.label}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

export function Landing() {
  const [hovered, setHovered] = useState(false);
  const [pos, setPos] = useState({ x: 0, y: 0 });

  const handleMouseMove = useCallback((e: React.MouseEvent<HTMLElement>) => {
    const rect = e.currentTarget.getBoundingClientRect();
    const PAD = 60;
    setPos({
      x: Math.max(PAD, Math.min(rect.width - PAD, e.clientX - rect.left)),
      y: Math.max(PAD, Math.min(rect.height - PAD, e.clientY - rect.top)),
    });
    setHovered(true);
  }, []);

  const handleMouseLeave = useCallback(() => setHovered(false), []);

  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      <style>{`
        @keyframes fadeUp {
          from { opacity: 0; transform: translateY(14px); }
          to   { opacity: 1; transform: translateY(0); }
        }
        .fade-up { animation: fadeUp 0.55s ease forwards; opacity: 0; }
        .delay-1 { animation-delay: 0.05s; }
        .delay-2 { animation-delay: 0.15s; }
        .delay-3 { animation-delay: 0.27s; }
        .delay-4 { animation-delay: 0.40s; }
        .hero-grid {
          background-image:
            linear-gradient(var(--chart-grid) 1px, transparent 1px),
            linear-gradient(90deg, var(--chart-grid) 1px, transparent 1px);
          background-size: 52px 52px;
        }
        .feature-card:hover .feature-index { color: hsl(var(--primary)); }
        .glass-nav { background: oklch(1 0 0 / 0.65); }
        .dark .glass-nav { background: oklch(0.141 0.005 285.823 / 0.60); }
      `}</style>

      {/* Nav */}
      <header className="glass-nav sticky top-0 z-50 flex h-16 items-center justify-between border-b px-6 md:px-16" style={{ backdropFilter: 'blur(16px) saturate(180%)' }}>
        <div className="flex items-center gap-2.5">
          <Navigation className="h-4 w-4 text-primary" />
          <span className="text-sm font-semibold tracking-tight" style={displayFont}>
            InvestPilot
          </span>
        </div>
        <nav className="flex items-center gap-1">
          <ThemeToggle />
          <Button variant="ghost" size="sm" asChild className="text-muted-foreground hover:text-foreground">
            <Link to="/login">Log in</Link>
          </Button>
          <Button size="sm" asChild>
            <Link to="/register" className="flex items-center gap-1.5">
              Get started
              <ArrowRight className="h-3.5 w-3.5" />
            </Link>
          </Button>
        </nav>
      </header>

      {/* Hero */}
      <section
        className="relative overflow-hidden border-b"
        onMouseMove={handleMouseMove}
        onMouseLeave={handleMouseLeave}
      >
        {/* Base grid — always visible at low opacity */}
        <div className="hero-grid absolute inset-0 opacity-[0.28]" />
        {/* Cursor spotlight — radial mask reveals grid at higher opacity near cursor */}
        <div
          className="hero-grid absolute inset-0"
          style={{
            opacity: hovered ? 0.78 : 0,
            transition: 'opacity 0.5s ease',
            maskImage: `radial-gradient(ellipse 480px 380px at ${pos.x}px ${pos.y}px, black 0%, transparent 72%)`,
            WebkitMaskImage: `radial-gradient(ellipse 480px 380px at ${pos.x}px ${pos.y}px, black 0%, transparent 72%)`,
          }}
        />
        <div className="relative max-w-6xl mx-auto px-6 md:px-16 py-24 md:py-36 grid md:grid-cols-[1fr_auto] gap-16 items-center">

          <div className="space-y-8">
            <div className="fade-up delay-1">
              <Badge
                variant="outline"
                className="px-3 py-1 text-[10px] font-medium tracking-[0.15em] uppercase text-muted-foreground"
              >
                Algorithmic Wealth Management
              </Badge>
            </div>

            <div className="fade-up delay-2 space-y-4">
              <h1
                className="text-5xl md:text-6xl lg:text-7xl leading-[1.05] tracking-tight"
                style={displayFont}
              >
                Intelligent portfolios,
                <br />
                <span className="text-primary italic">
                  scientifically{' '}engineered.
                </span>
              </h1>
              <p className="text-base text-muted-foreground max-w-lg leading-relaxed">
                InvestPilot builds and rebalances your ETF portfolio using Hierarchical Risk
                Parity. Define your risk tolerance once and let the algorithm handle the rest.
              </p>
            </div>

            <div className="fade-up delay-3 flex flex-wrap gap-3">
              <Button size="lg" asChild>
                <Link to="/register" className="flex items-center gap-2">
                  Start investing
                  <ArrowRight className="h-4 w-4" />
                </Link>
              </Button>
              <Button size="lg" variant="outline" asChild>
                <Link to="/login">Log in to your account</Link>
              </Button>
            </div>

            <div className="fade-up delay-4 pt-2 max-w-sm">
              <p className="text-[10px] text-muted-foreground/50 uppercase tracking-[0.12em] mb-3 font-mono">
                Sample allocations
              </p>
              <AllocationBar />
            </div>
          </div>

          {/* Decorative ticker */}
          <div className="hidden md:flex flex-col gap-2.5 opacity-[0.40] select-none w-28">
            {tickerItems.map((t) => (
              <div key={t.symbol} className="flex items-center justify-between gap-3 font-mono text-[10px]">
                <span className="text-foreground font-medium">{t.symbol}</span>
                <span className={t.positive ? 'text-green-500' : 'text-red-400'}>{t.change}</span>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Stats */}
      <div className="border-b bg-card/50">
        <div className="max-w-6xl mx-auto grid grid-cols-2 md:grid-cols-4 divide-x divide-y md:divide-y-0 divide-border">
          {stats.map((s) => (
            <div key={s.index} className="flex flex-col py-10 px-8 gap-1.5">
              <span className="text-[10px] text-muted-foreground/40 font-mono tracking-widest">
                {s.index}
              </span>
              <span
                className="text-3xl md:text-4xl font-bold text-primary leading-none"
                style={displayFont}
              >
                {s.value}
              </span>
              <span className="text-xs text-muted-foreground leading-snug max-w-[140px] mt-0.5">
                {s.label}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Features */}
      <section className="py-24 px-6 md:px-16">
        <div className="max-w-6xl mx-auto">
          <div className="mb-16 space-y-3 max-w-xl">
            <p className="text-[10px] text-muted-foreground/50 font-mono tracking-[0.15em] uppercase">
              Platform capabilities
            </p>
            <h2
              className="text-3xl md:text-4xl text-foreground leading-tight"
              style={displayFont}
            >
              Built for serious investors.
            </h2>
            <p className="text-muted-foreground text-sm leading-relaxed">
              Every component is designed around quantitative finance principles, not simplified
              heuristics.
            </p>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-px border rounded-xl overflow-hidden bg-border">
            {features.map((f) => (
              <div
                key={f.title}
                className="feature-card bg-card p-8 space-y-4 hover:bg-card/70 transition-colors group"
              >
                <div className="flex items-start justify-between">
                  <div className="rounded-lg bg-primary/10 p-2.5 group-hover:bg-primary/15 transition-colors">
                    <f.icon className="h-4 w-4 text-primary" />
                  </div>
                  <span className="feature-index text-xs font-mono text-muted-foreground/25 transition-colors">
                    {f.index}
                  </span>
                </div>
                <div className="space-y-2">
                  <h3 className="font-semibold text-sm text-foreground tracking-tight">
                    {f.title}
                  </h3>
                  <p className="text-sm text-muted-foreground leading-relaxed">{f.description}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="border-t border-b bg-card">
        <div className="max-w-6xl mx-auto px-6 md:px-16 py-20 flex flex-col md:flex-row items-center justify-between gap-8">
          <div className="space-y-2 text-center md:text-left">
            <h2
              className="text-2xl md:text-3xl font-semibold leading-tight"
              style={displayFont}
            >
              Ready to put your capital to work?
            </h2>
            <p className="text-muted-foreground text-sm">
              Create an account in under a minute. No minimum deposit in paper trading mode.
            </p>
          </div>
          <Button size="lg" asChild className="flex-shrink-0">
            <Link to="/register" className="flex items-center gap-2">
              Open an account
              <ArrowRight className="h-4 w-4" />
            </Link>
          </Button>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-6 px-6 md:px-16">
        <div className="max-w-6xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-muted-foreground">
          <div className="flex items-center gap-2">
            <Navigation className="h-3.5 w-3.5" />
            <span style={displayFont} className="font-medium">
              InvestPilot
            </span>
          </div>
          <Separator orientation="vertical" className="h-4 hidden sm:block" />
          <span className="text-muted-foreground/60">
            Paper trading environment. Not financial advice.
          </span>
          <div className="flex items-center gap-4">
            <Link to="/login" className="hover:text-foreground transition-colors">
              Log in
            </Link>
            <Link to="/register" className="hover:text-foreground transition-colors">
              Register
            </Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
