import { useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  LineChart as LineChartIcon,
  Line,
  XAxis,
  YAxis,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  BarChart,
  Bar,
  Cell,
  CartesianGrid,
} from "recharts";
import { format, parseISO } from "date-fns";
import {
  Wallet,
  TrendingUp,
  TrendingDown,
  LineChart as LineChartIcon2,
  ArrowRight,
  ClipboardList,
  PlusCircle,
  MinusCircle,
  BarChart3,
  Info,
} from "lucide-react";

import { useAuthStore } from "@/stores/authStore";
import { portfolioApi } from "@/api/portfolio";

import { DepositDialog } from "@/components/transactions/DepositDialog";
import { StripeDepositDialog } from "@/components/transactions/StripeDepositDialog";
import { CashoutDialog } from "@/components/transactions/CashoutDialog";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip as UITooltip,
  TooltipContent as UITooltipContent,
  TooltipProvider as UITooltipProvider,
  TooltipTrigger as UITooltipTrigger,
} from "@/components/ui/tooltip";

function formatUSD(value: number) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
  }).format(value);
}

function formatPct(value: number) {
  const sign = value >= 0 ? "+" : "";
  return `${sign}${value.toFixed(2)}%`;
}

const riskLabels: Record<number, string> = {
  1: "Very Conservative",
  2: "Conservative",
  3: "Balanced",
  4: "Growth",
  5: "Aggressive Growth",
};

function MiniPerfTooltip({ active, payload, label }: any) {
  if (!active || !payload?.length) return null;
  const val: number = payload[0]?.value ?? 0;
  const isGain = val >= 0;
  const date = label ? format(parseISO(label), "PPP") : "";
  return (
    <div className="rounded-md border bg-popover px-2.5 py-1.5 shadow-md text-xs space-y-0.5 min-w-[140px]">
      <p className="text-muted-foreground">{date}</p>
      <p className={`font-mono font-semibold ${isGain ? "text-emerald-400" : "text-red-400"}`}>
        {isGain ? "+" : ""}{val.toFixed(2)}%
      </p>
    </div>
  );
}

function MiniPerformanceChart() {
  const { data, isLoading } = useQuery({
    queryKey: ["portfolio-history", "1M"],
    queryFn: () => portfolioApi.getHistory("1M").then((res) => res.data),
    staleTime: 60_000,
  });

  const points = data?.data ?? [];
  const hasData = points.length > 0;

  if (isLoading) {
    return <div className="h-[260px] animate-pulse rounded-lg bg-muted/30" />;
  }

  if (!hasData) {
    return (
      <div className="flex flex-col items-center justify-center h-[260px] rounded-xl border border-dashed border-border bg-muted/10 text-center gap-2">
        <LineChartIcon2 className="h-6 w-6 text-muted-foreground/40" />
        <p className="text-xs text-muted-foreground">No performance data yet</p>
      </div>
    );
  }

  const last = points[points.length - 1]?.return_percentage ?? 0;
  const isGain = last >= 0;
  const lineColor = isGain ? "#10b981" : "#ef4444";

  return (
    <div className="space-y-2">
      <div className="flex items-baseline gap-2">
        <span className={`text-lg font-bold font-mono tracking-tight ${isGain ? "text-emerald-500" : "text-red-500"}`}>
          {isGain ? "+" : ""}{last.toFixed(2)}%
        </span>
        <span className="text-xs text-muted-foreground">past 30 days</span>
      </div>
      <ResponsiveContainer width="100%" height={220}>
        <LineChartIcon data={points} margin={{ top: 4, right: 4, left: -20, bottom: 0 }}>
          <XAxis dataKey="timestamp" tick={false} axisLine={false} tickLine={false} />
          <YAxis
            tickFormatter={(v) => `${v > 0 ? "+" : ""}${v.toFixed(0)}%`}
            tick={{ fontSize: 10, fill: "var(--muted-foreground)" }}
            axisLine={false}
            tickLine={false}
            width={48}
          />
          <Tooltip content={<MiniPerfTooltip />} cursor={{ stroke: "var(--muted-foreground)", strokeWidth: 1, strokeOpacity: 0.4 }} />
          <ReferenceLine y={0} stroke="var(--muted-foreground)" strokeWidth={1} strokeDasharray="3 3" strokeOpacity={0.5} />
          <Line
            type="monotone"
            dataKey="return_percentage"
            stroke={lineColor}
            strokeWidth={1.5}
            dot={false}
            activeDot={{ r: 3, fill: lineColor }}
          />
        </LineChartIcon>
      </ResponsiveContainer>
    </div>
  );
}

const ALLOC_COLORS = [
  "var(--chart-1)",
  "var(--chart-2)",
  "var(--chart-3)",
  "var(--chart-4)",
  "var(--chart-5)",
];

function MiniAllocTooltip({ active, payload, totalValue }: any) {
  if (!active || !payload?.length) return null;
  const d = payload[0]?.payload;
  const pct = totalValue > 0 ? (d.value / totalValue) * 100 : 0;
  return (
    <div className="rounded-md border bg-popover px-2.5 py-1.5 shadow-md text-xs space-y-0.5 min-w-[120px]">
      <p className="font-semibold">{d.name}</p>
      <p className="text-muted-foreground font-mono">{pct.toFixed(1)}%</p>
    </div>
  );
}

function MiniAllocationBar() {
  const { data, isLoading } = useQuery({
    queryKey: ["portfolio-allocation"],
    queryFn: () => portfolioApi.getPortfolio().then((res) => res.data),
    staleTime: 60_000,
  });

  const holdings = data?.holdings ?? [];
  const totalValue = data?.live_total_value ?? 0;

  const grouped = holdings.reduce((acc: any, h: any) => {
    if (!acc[h.ticker]) acc[h.ticker] = { name: h.ticker, value: 0 };
    acc[h.ticker].value += h.current_value;
    return acc;
  }, {});

  const barData: any[] = Object.values(grouped);
  barData.sort((a: any, b: any) => {
    if (a.name === "USD") return 1;
    if (b.name === "USD") return -1;
    return b.value - a.value;
  });

  if (isLoading) return <div className="h-[220px] animate-pulse rounded-lg bg-muted/30" />;

  if (barData.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-[220px] rounded-xl border border-dashed border-border bg-muted/10 text-center gap-2">
        <BarChart3 className="h-6 w-6 text-muted-foreground/40" />
        <p className="text-xs text-muted-foreground">No allocation data yet</p>
      </div>
    );
  }

  return (
    <ResponsiveContainer width="100%" height={220}>
      <BarChart data={barData} margin={{ top: 4, right: 4, left: 4, bottom: 0 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--muted-foreground)" strokeOpacity={0.2} vertical={false} />
        <XAxis
          dataKey="name"
          tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
          tickLine={false}
          axisLine={false}
          dy={6}
        />
        <YAxis
          tickFormatter={(v) => `$${v >= 1000 ? `${(v / 1000).toFixed(0)}k` : v}`}
          tick={{ fontSize: 10, fill: "var(--muted-foreground)" }}
          tickLine={false}
          axisLine={false}
          width={44}
        />
        <Tooltip content={<MiniAllocTooltip totalValue={totalValue} />} cursor={{ fill: "var(--muted)", opacity: 0.3 }} />
        <Bar dataKey="value" radius={[3, 3, 0, 0]} maxBarSize={48}>
          {barData.map((entry: any, index: number) =>
            <Cell key={`cell-${index}`} fill={entry.name === "USD" ? "var(--muted)" : ALLOC_COLORS[index % ALLOC_COLORS.length]} />
          )}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  );
}

export function Dashboard() {
  const { user } = useAuthStore();
  const [paperDepositOpen, setPaperDepositOpen] = useState(false);
  const [stripeDepositOpen, setStripeDepositOpen] = useState(false);
  const [cashoutOpen, setCashoutOpen] = useState(false);

  const hasProfile = user && user.risk_tolerance > 0 && user.investment_horizon > 0;

  const { data: portfolio, isLoading: portfolioLoading } = useQuery({
    queryKey: ["portfolio-allocation"],
    queryFn: () => portfolioApi.getPortfolio().then((res) => res.data),
    staleTime: 60_000,
  });

  const liveTotal = portfolio?.live_total_value ?? 0;
  const netContributions = portfolio?.net_contributions ?? 0;
  const allTimePL = portfolio?.all_time_profit_loss ?? 0;
  const allTimePct = netContributions > 0 ? (allTimePL / netContributions) * 100 : 0;
  const isGain = allTimePL >= 0;
  const hasPortfolio = portfolio?.holdings && portfolio.holdings.length > 0;

  return (
    <div className="p-6 md:p-8 space-y-6 max-w-6xl mx-auto">

      {/* Page header */}
      <div className="space-y-0.5">
        <h1 className="text-xl font-semibold tracking-tight">Dashboard</h1>
        <div className="flex items-center gap-1.5">
          <p className="text-sm text-muted-foreground">
            {user?.email ? `Signed in as ${user.email.split("@")[0]}` : "Overview of your account"}
          </p>
          <UITooltipProvider delayDuration={200}>
            <UITooltip>
              <UITooltipTrigger asChild>
                <Info className="h-3 w-3 text-muted-foreground/40 cursor-default shrink-0 pointer-events-auto" />
              </UITooltipTrigger>
              <UITooltipContent side="right" className="text-xs border border-border/50 bg-popover text-popover-foreground shadow-md">
                Values reflect latest market data, refreshed every 15 minutes during trading hours.
              </UITooltipContent>
            </UITooltip>
          </UITooltipProvider>
        </div>
      </div>

      {/* Onboarding callout */}
      {!hasProfile && (
        <Alert className="border-amber-500/30 bg-amber-500/5">
          <ClipboardList className="h-4 w-4 text-amber-500" />
          <AlertTitle className="text-amber-600 dark:text-amber-400 font-medium">
            Complete your investment profile
          </AlertTitle>
          <AlertDescription className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mt-1">
            <span className="text-sm text-muted-foreground">
              Answer a few questions so we can build a personalized HRP portfolio for you.
            </span>
            <Button asChild size="sm" className="shrink-0 gap-1.5">
              <Link to="/onboarding">
                Start questionnaire
                <ArrowRight className="h-3.5 w-3.5" />
              </Link>
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {/* KPI grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">

        {/* Wallet */}
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Wallet Balance</p>
              <Wallet className="h-4 w-4 text-muted-foreground/50" />
            </div>
            <CardTitle className="text-2xl font-bold tracking-tight">
              {formatUSD(user?.wallet_balance ?? 0)}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <Separator />
            <div className="grid grid-cols-2 gap-2">
              <Button size="sm" className="gap-1.5 text-xs h-8" onClick={() => setPaperDepositOpen(true)}>
                <PlusCircle className="h-3.5 w-3.5" />
                Deposit
              </Button>
              <Button size="sm" variant="secondary" className="gap-1.5 text-xs h-8" onClick={() => setStripeDepositOpen(true)}>
                <PlusCircle className="h-3.5 w-3.5" />
                Stripe
              </Button>
              <Button size="sm" variant="outline" className="col-span-2 gap-1.5 text-xs h-8" onClick={() => setCashoutOpen(true)}>
                <MinusCircle className="h-3.5 w-3.5" />
                Cashout
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Portfolio */}
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Portfolio Value</p>
              <div className="flex items-center gap-1.5">
                {hasPortfolio && (
                  <Badge
                    variant="outline"
                    className={`text-xs h-5 px-1.5 ${
                      isGain
                        ? "border-emerald-500/30 text-emerald-600 dark:text-emerald-400"
                        : "border-red-500/30 text-red-600 dark:text-red-400"
                    }`}
                  >
                    {netContributions > 0 ? formatPct(allTimePct) : "—"}
                  </Badge>
                )}
                {isGain
                  ? <TrendingUp className="h-4 w-4 text-emerald-500" />
                  : <TrendingDown className="h-4 w-4 text-red-500" />
                }
              </div>
            </div>
            {portfolioLoading ? (
              <Skeleton className="h-8 w-32 mt-1" />
            ) : (
              <CardTitle className="text-2xl font-bold tracking-tight">
                {hasPortfolio ? formatUSD(liveTotal) : "N/A"}
              </CardTitle>
            )}
          </CardHeader>
          <CardContent className="space-y-3">
            <Separator />
            {portfolioLoading ? (
              <div className="space-y-2">
                <Skeleton className="h-3.5 w-full" />
                <Skeleton className="h-3.5 w-3/4" />
              </div>
            ) : hasPortfolio ? (
              <div className="space-y-2 text-xs">
                <div className="flex justify-between items-center">
                  <span className="text-muted-foreground">Net contributions</span>
                  <span className="font-mono font-medium">{formatUSD(netContributions)}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-muted-foreground">All-time P&L</span>
                  <span className={`font-mono font-semibold ${isGain ? "text-emerald-500" : "text-red-500"}`}>
                    {isGain ? "+" : ""}{formatUSD(allTimePL)}
                  </span>
                </div>
              </div>
            ) : (
              <p className="text-xs text-muted-foreground leading-relaxed">
                No active investments yet. Deposit funds and visit Portfolio to get started.
              </p>
            )}
            <Button asChild size="sm" variant="outline" className="w-full gap-1.5 text-xs h-8 mt-2">
              <Link to="/portfolio">
                View portfolio
                <ArrowRight className="h-3.5 w-3.5" />
              </Link>
            </Button>
          </CardContent>
        </Card>

        {/* Forecast teaser */}
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Forecast Engine</p>
              <LineChartIcon2 className="h-4 w-4 text-muted-foreground/50" />
            </div>
            <CardTitle className="text-base font-semibold tracking-tight">Monte Carlo Simulator</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <CardDescription className="text-xs leading-relaxed">
              Project your portfolio across 10,000 market scenarios. Personalized to your risk
              profile{hasProfile ? ` (${riskLabels[user!.risk_tolerance]})` : ""}.
            </CardDescription>
            <Separator />
            <Button asChild size="sm" className="w-full gap-1.5 text-xs h-8">
              <Link to="/forecast">
                Launch forecaster
                <ArrowRight className="h-3.5 w-3.5" />
              </Link>
            </Button>
          </CardContent>
        </Card>
      </div>

      {/* Allocation + Performance */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <div className="space-y-0.5">
              <CardTitle className="text-sm font-semibold tracking-tight">Asset Allocation</CardTitle>
              <CardDescription className="text-xs">Current distribution of your portfolio</CardDescription>
            </div>
            <Button asChild variant="ghost" size="sm" className="gap-1.5 text-xs shrink-0 h-8">
              <Link to="/portfolio">
                View details
                <ArrowRight className="h-3.5 w-3.5" />
              </Link>
            </Button>
          </CardHeader>
          <CardContent>
            <MiniAllocationBar />
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <div className="space-y-0.5">
              <CardTitle className="text-sm font-semibold tracking-tight">Performance</CardTitle>
              <CardDescription className="text-xs">Portfolio return over time</CardDescription>
            </div>
            <Button asChild variant="ghost" size="sm" className="gap-1.5 text-xs shrink-0 h-8">
              <Link to="/portfolio">
                View details
                <ArrowRight className="h-3.5 w-3.5" />
              </Link>
            </Button>
          </CardHeader>
          <CardContent>
            <MiniPerformanceChart />
          </CardContent>
        </Card>
      </div>

      {/* Dialogs */}
      <DepositDialog open={paperDepositOpen} onOpenChange={setPaperDepositOpen} />
      <StripeDepositDialog open={stripeDepositOpen} onOpenChange={setStripeDepositOpen} />
      <CashoutDialog open={cashoutOpen} onOpenChange={setCashoutOpen} />
    </div>
  );
}
