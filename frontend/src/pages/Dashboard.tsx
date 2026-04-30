import { useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  Wallet,
  TrendingUp,
  TrendingDown,
  LineChart,
  ArrowRight,
  ClipboardList,
  PlusCircle,
  MinusCircle,
} from "lucide-react";

import { useAuthStore } from "@/stores/authStore";
import { portfolioApi } from "@/api/portfolio";

import { AllocationPie } from "@/components/charts/AllocationPie";
import { DepositDialog } from "@/components/transactions/DepositDialog";
import { StripeDepositDialog } from "@/components/transactions/StripeDepositDialog";
import { CashoutDialog } from "@/components/transactions/CashoutDialog";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";

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
        <p className="text-sm text-muted-foreground">
          {user?.email ? `Signed in as ${user.email}` : "Overview of your account"}
        </p>
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
            <Button asChild size="sm" variant="outline" className="w-full gap-1.5 text-xs h-8">
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
              <LineChart className="h-4 w-4 text-muted-foreground/50" />
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

      {/* Allocation preview */}
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
          <AllocationPie showTitle={false} />
        </CardContent>
      </Card>

      {/* Dialogs */}
      <DepositDialog open={paperDepositOpen} onOpenChange={setPaperDepositOpen} />
      <StripeDepositDialog open={stripeDepositOpen} onOpenChange={setStripeDepositOpen} />
      <CashoutDialog open={cashoutOpen} onOpenChange={setCashoutOpen} />
    </div>
  );
}
