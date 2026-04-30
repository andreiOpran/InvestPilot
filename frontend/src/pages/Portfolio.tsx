import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { TrendingUp, TrendingDown, DollarSign, BarChart3, LineChart } from "lucide-react";

import { portfolioApi } from "@/api/portfolio";
import { ValueOverTime } from "@/components/charts/ValueOverTime";
import { PerformanceChart } from "@/components/charts/PerformanceChart";
import { AllocationPie } from "@/components/charts/AllocationPie";
import { TransactionTable } from "@/components/portfolio/TransactionTable";
import { InvestDialog } from "@/components/transactions/InvestDialog";
import { SellDialog } from "@/components/transactions/SellDialog";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

function formatUSD(value: number) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
}

function formatPct(value: number) {
  const sign = value >= 0 ? "+" : "";
  return `${sign}${value.toFixed(2)}%`;
}

interface StatCardProps {
  label: string;
  value: string;
  sub?: string;
  subPositive?: boolean;
  icon: React.ReactNode;
  loading: boolean;
}

function StatCard({ label, value, sub, subPositive, icon, loading }: StatCardProps) {
  return (
    <Card>
      <CardContent className="pt-2 pb-2">
        {loading ? (
          <div className="space-y-2">
            <Skeleton className="h-3 w-20" />
            <Skeleton className="h-6 w-28" />
            <Skeleton className="h-3 w-16" />
          </div>
        ) : (
          <div className="space-y-1">
            <div className="flex items-center justify-between">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">{label}</p>
              <span className="text-muted-foreground/40">{icon}</span>
            </div>
            <p className="text-xl font-bold tracking-tight">{value}</p>
            {sub !== undefined && (
              <p className={`text-xs font-medium ${
                subPositive === undefined
                  ? "text-muted-foreground"
                  : subPositive
                  ? "text-emerald-500"
                  : "text-red-500"
              }`}>
                {sub}
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export function Portfolio() {
  const [investOpen, setInvestOpen] = useState(false);
  const [sellOpen, setSellOpen] = useState(false);
  const [chartTab, setChartTab] = useState<"value" | "performance">("value");

  const { data, isLoading } = useQuery({
    queryKey: ["portfolio-allocation"],
    queryFn: () => portfolioApi.getPortfolio().then((res) => res.data),
    staleTime: 60_000,
  });

  const liveTotal = data?.live_total_value ?? 0;
  const netContributions = data?.net_contributions ?? 0;
  const allTimePL = data?.all_time_profit_loss ?? 0;
  const allTimePct = netContributions > 0 ? (allTimePL / netContributions) * 100 : 0;
  const isGain = allTimePL >= 0;
  const hasHoldings = Boolean(data?.holdings?.length);

  return (
    <div className="p-6 md:p-8 space-y-6 max-w-7xl mx-auto">

      {/* Header */}
      <div className="space-y-0.5">
        <h1 className="text-xl font-semibold tracking-tight">Portfolio</h1>
        <p className="text-sm text-muted-foreground">Live overview of your active investment round</p>
      </div>

      {/* Stats + action strip */}
      <div className="grid grid-cols-2 lg:grid-cols-5 gap-4">
        <StatCard
          label="Portfolio Value"
          value={formatUSD(liveTotal)}
          icon={<DollarSign className="h-3.5 w-3.5" />}
          loading={isLoading}
        />
        <StatCard
          label="Net Contributions"
          value={formatUSD(netContributions)}
          icon={<BarChart3 className="h-3.5 w-3.5" />}
          loading={isLoading}
        />
        <StatCard
          label="All-Time P&L"
          value={formatUSD(allTimePL)}
          sub={netContributions > 0 ? formatPct(allTimePct) : undefined}
          subPositive={netContributions > 0 ? isGain : undefined}
          icon={isGain
            ? <TrendingUp className="h-3.5 w-3.5 text-emerald-500" />
            : <TrendingDown className="h-3.5 w-3.5 text-red-500" />
          }
          loading={isLoading}
        />
        <StatCard
          label="All-Time Return"
          value={netContributions > 0 ? formatPct(allTimePct) : "—"}
          sub={netContributions > 0 ? `on ${formatUSD(netContributions)} invested` : "No investments yet"}
          subPositive={netContributions > 0 ? isGain : undefined}
          icon={isGain
            ? <TrendingUp className="h-3.5 w-3.5 text-emerald-500" />
            : <TrendingDown className="h-3.5 w-3.5 text-red-500" />
          }
          loading={isLoading}
        />

        {/* Action card */}
        <Card className="flex flex-col col-span-2 lg:col-span-1">
          <CardContent className="flex-1 flex flex-col justify-center gap-2 pt-2 pb-2">
            <Button
              className="w-full h-9 text-xs font-medium gap-1.5"
              onClick={() => setInvestOpen(true)}
            >
              <TrendingUp className="h-3.5 w-3.5" />
              Invest
            </Button>
            <Button
              variant="outline"
              className="w-full h-9 text-xs font-medium gap-1.5 border-red-500/30 text-red-600 hover:bg-red-500/10 hover:text-red-600 dark:text-red-400"
              onClick={() => setSellOpen(true)}
              disabled={!hasHoldings}
            >
              <TrendingDown className="h-3.5 w-3.5" />
              Sell
            </Button>
          </CardContent>
        </Card>
      </div>

      {/* Allocation + Charts */}
      <div className="grid grid-cols-1 xl:grid-cols-5 gap-6">
        {/* Allocation pie — narrower column */}
        <Card className="xl:col-span-2">
          <CardContent className="pt-6">
            <AllocationPie />
          </CardContent>
        </Card>

        {/* Performance charts — wider column */}
        <Card className="xl:col-span-3">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <div>
                {chartTab === "value" ? (
                  <>
                    <h3 className="text-lg font-semibold">Portfolio Value Over Time</h3>
                    <p className="text-sm text-muted-foreground">Absolute market value of your active portfolio</p>
                  </>
                ) : (
                  <>
                    <h3 className="text-lg font-semibold">Performance</h3>
                    <p className="text-sm text-muted-foreground">Return since the start of the selected range</p>
                  </>
                )}
              </div>
              <div className="flex bg-muted/50 p-1 rounded-lg">
                <Button
                  variant={chartTab === "value" ? "secondary" : "ghost"}
                  size="sm"
                  onClick={() => setChartTab("value")}
                  className="h-8 px-3"
                >
                  <LineChart className="h-4 w-4 mr-2" />
                  Value
                </Button>
                <Button
                  variant={chartTab === "performance" ? "secondary" : "ghost"}
                  size="sm"
                  onClick={() => setChartTab("performance")}
                  className="h-8 px-3"
                >
                  <TrendingUp className="h-4 w-4 mr-2" />
                  Performance
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent className="pt-2">
            {chartTab === "value"
              ? <ValueOverTime onInvestClick={() => setInvestOpen(true)} />
              : <PerformanceChart onInvestClick={() => setInvestOpen(true)} />
            }
          </CardContent>
        </Card>
      </div>

      {/* Transaction history */}
      <Card>
        <CardContent className="pt-6">
          <TransactionTable />
        </CardContent>
      </Card>

      <InvestDialog open={investOpen} onOpenChange={setInvestOpen} />
      <SellDialog open={sellOpen} onOpenChange={setSellOpen} portfolioValue={liveTotal} />
    </div>
  );
}
