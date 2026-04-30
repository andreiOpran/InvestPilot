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

  const hasProfile =
    user && user.risk_tolerance > 0 && user.investment_horizon > 0;

  const { data: portfolio, isLoading: portfolioLoading } = useQuery({
    queryKey: ["portfolio-allocation"],
    queryFn: () => portfolioApi.getPortfolio().then((res) => res.data),
    staleTime: 60_000,
  });

  const liveTotal = portfolio?.live_total_value ?? 0;
  const netContributions = portfolio?.net_contributions ?? 0;
  const allTimePL = portfolio?.all_time_profit_loss ?? 0;
  const allTimePct =
    netContributions > 0 ? (allTimePL / netContributions) * 100 : 0;
  const isGain = allTimePL >= 0;
  const hasPortfolio = portfolio?.holdings && portfolio.holdings.length > 0;

  return (
    <div className="p-6 space-y-6 max-w-7xl mx-auto">
      {/* Page header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground text-sm mt-0.5">
          Welcome back{user?.email ? `, ${user.email}` : ""}.
        </p>
      </div>

      {/* Onboarding callout */}
      {!hasProfile && (
        <Alert className="border-amber-500/40 bg-amber-500/5">
          <ClipboardList className="h-4 w-4 text-amber-500" />
          <AlertTitle className="text-amber-600 dark:text-amber-400">
            Complete your investment profile
          </AlertTitle>
          <AlertDescription className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mt-1">
            <span className="text-sm text-muted-foreground">
              Answer a few questions so we can build a personalized HRP
              portfolio for you.
            </span>
            <Button asChild size="sm" className="shrink-0 gap-2">
              <Link to="/onboarding">
                Start Questionnaire
                <ArrowRight className="h-3.5 w-3.5" />
              </Link>
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {/* Three-column overview grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {/* Wallet Card */}
        <Card className="flex flex-col">
          <CardHeader className="pb-3">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Wallet className="h-4 w-4" />
              Wallet Balance
            </div>
            <CardTitle className="text-3xl font-bold tracking-tight">
              {formatUSD(user?.wallet_balance ?? 0)}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex-1 space-y-3">
            <Separator />
            <div className="grid grid-cols-2 gap-2">
              <Button
                size="sm"
                className="gap-1.5 text-xs"
                onClick={() => setPaperDepositOpen(true)}
              >
                <PlusCircle className="h-3.5 w-3.5" />
                Deposit
              </Button>
              <Button
                size="sm"
                variant="secondary"
                className="gap-1.5 text-xs"
                onClick={() => setStripeDepositOpen(true)}
              >
                <PlusCircle className="h-3.5 w-3.5" />
                Deposit (Stripe)
              </Button>
              <Button
                size="sm"
                variant="outline"
                className="col-span-2 gap-1.5 text-xs"
                onClick={() => setCashoutOpen(true)}
              >
                <MinusCircle className="h-3.5 w-3.5" />
                Cashout
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Portfolio Snapshot Card */}
        <Card className="flex flex-col">
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                {isGain ? (
                  <TrendingUp className="h-4 w-4 text-emerald-500" />
                ) : (
                  <TrendingDown className="h-4 w-4 text-red-500" />
                )}
                Portfolio Value
              </div>
              {hasPortfolio && (
                <Badge
                  variant="outline"
                  className={`text-xs ${
                    isGain
                      ? "border-emerald-500/40 text-emerald-600 dark:text-emerald-400"
                      : "border-red-500/40 text-red-600 dark:text-red-400"
                  }`}
                >
                  {netContributions > 0 ? formatPct(allTimePct) : "—"}
                </Badge>
              )}
            </div>
            {portfolioLoading ? (
              <Skeleton className="h-9 w-36 mt-1" />
            ) : (
              <CardTitle className="text-3xl font-bold tracking-tight">
                {hasPortfolio ? formatUSD(liveTotal) : "N/A"}
              </CardTitle>
            )}
          </CardHeader>
          <CardContent className="flex-1 space-y-3">
            <Separator />
            {portfolioLoading ? (
              <div className="space-y-2">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
              </div>
            ) : hasPortfolio ? (
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Net Contributions</span>
                  <span className="font-mono">{formatUSD(netContributions)}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">All-Time P&L</span>
                  <span
                    className={`font-mono font-medium ${
                      isGain ? "text-emerald-500" : "text-red-500"
                    }`}
                  >
                    {isGain ? "+" : ""}
                    {formatUSD(allTimePL)}
                  </span>
                </div>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                No active investments yet. Deposit funds and go to Full Portfolio to
                get started.
              </p>
            )}
            <Button asChild size="sm" className="w-full gap-1.5 text-xs mt-auto">
              <Link to="/portfolio">
                View Full Portfolio
                <ArrowRight className="h-3.5 w-3.5" />
              </Link>
            </Button>
          </CardContent>
        </Card>

        {/* Forecast Teaser Card */}
        <Card className="flex flex-col md:col-span-2 lg:col-span-1">
          <CardHeader className="pb-3">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <LineChart className="h-4 w-4" />
              Forecast Engine
            </div>
            <CardTitle className="text-lg font-semibold">
              Monte Carlo Simulator
            </CardTitle>
          </CardHeader>
          <CardContent className="flex-1 space-y-3">
            <CardDescription>
              Project your portfolio's future value across 10,000 market
              scenarios. Personalized to your risk profile
              {hasProfile ? ` (${riskLabels[user!.risk_tolerance]})` : ""}.
            </CardDescription>
            <Separator />
            {/* <Button asChild variant="ghost" size="sm" className="w-full gap-1.5 text-xs mt-auto"></Button> */}
            <Button asChild size="sm" className="w-full gap-1.5 text-xs mt-auto">
              <Link to="/forecast">
                Launch Forecaster
                <ArrowRight className="h-4 w-4" />
              </Link>
            </Button>
          </CardContent>
        </Card>
      </div>

      {/* Allocation Preview */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <div>
            <CardTitle className="text-base font-semibold">Asset Allocation</CardTitle>
            <CardDescription className="text-xs mt-0.5">
              Current distribution of your active portfolio
            </CardDescription>
          </div>
          <Button asChild variant="ghost" size="sm" className="gap-1.5 text-xs shrink-0">
            <Link to="/portfolio">
              View Details
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
