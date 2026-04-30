import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { TrendingUp, TrendingDown, DollarSign, BarChart3 } from "lucide-react";

import { portfolioApi } from "@/api/portfolio";
import { ValueOverTime } from "@/components/charts/ValueOverTime";
import { PerformanceChart } from "@/components/charts/PerformanceChart";
import { AllocationPie } from "@/components/charts/AllocationPie";
import { TransactionTable } from "@/components/portfolio/TransactionTable";
import { InvestDialog } from "@/components/transactions/InvestDialog";
import { SellDialog } from "@/components/transactions/SellDialog";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

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
      <CardContent className="pt-6">
        {loading ? (
          <div className="space-y-2">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-7 w-32" />
            <Skeleton className="h-3 w-20" />
          </div>
        ) : (
          <div className="space-y-1">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              {icon}
              {label}
            </div>
            <p className="text-2xl font-bold tracking-tight">{value}</p>
            {sub !== undefined && (
              <p
                className={`text-xs font-medium ${
                  subPositive === undefined
                    ? "text-muted-foreground"
                    : subPositive
                    ? "text-emerald-500"
                    : "text-red-500"
                }`}
              >
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

  const { data, isLoading } = useQuery({
    queryKey: ["portfolio-allocation"],
    queryFn: () => portfolioApi.getPortfolio().then((res) => res.data),
    staleTime: 60_000,
  });

  const liveTotal = data?.live_total_value ?? 0;
  const netContributions = data?.net_contributions ?? 0;
  const allTimePL = data?.all_time_profit_loss ?? 0;
  const allTimePct =
    netContributions > 0 ? (allTimePL / netContributions) * 100 : 0;
  const isGain = allTimePL >= 0;

  return (
    <div className="p-6 space-y-6 max-w-7xl mx-auto">
      {/* Page header */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Portfolio</h1>
          <p className="text-muted-foreground text-sm mt-0.5">
            Live overview of your active investment round
          </p>
        </div>
        <div className="flex items-center gap-3 shrink-0">
          <Button
            variant="outline"
            size="lg"
            className="h-11 px-5 border-red-500/40 text-red-600 hover:bg-red-500/10 hover:text-red-600 dark:text-red-400 font-semibold"
            onClick={() => setSellOpen(true)}
            disabled={!data?.holdings?.length}
          >
            <TrendingDown className="h-4 w-4 mr-2" />
            Sell
          </Button>
          <Button size="lg" className="h-11 px-5 font-semibold" onClick={() => setInvestOpen(true)}>
            <TrendingUp className="h-4 w-4 mr-2" />
            Invest
          </Button>
        </div>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          label="Portfolio Value"
          value={formatUSD(liveTotal)}
          icon={<DollarSign className="h-4 w-4" />}
          loading={isLoading}
        />
        <StatCard
          label="Net Contributions"
          value={formatUSD(netContributions)}
          icon={<BarChart3 className="h-4 w-4" />}
          loading={isLoading}
        />
        <StatCard
          label="All-Time P&L"
          value={formatUSD(allTimePL)}
          sub={netContributions > 0 ? formatPct(allTimePct) : undefined}
          subPositive={netContributions > 0 ? isGain : undefined}
          icon={
            isGain ? (
              <TrendingUp className="h-4 w-4 text-emerald-500" />
            ) : (
              <TrendingDown className="h-4 w-4 text-red-500" />
            )
          }
          loading={isLoading}
        />
        <StatCard
          label="All-Time Return"
          value={netContributions > 0 ? formatPct(allTimePct) : "—"}
          sub={netContributions > 0 ? `on ${formatUSD(netContributions)} invested` : "No investments yet"}
          subPositive={netContributions > 0 ? isGain : undefined}
          icon={
            isGain ? (
              <TrendingUp className="h-4 w-4 text-emerald-500" />
            ) : (
              <TrendingDown className="h-4 w-4 text-red-500" />
            )
          }
          loading={isLoading}
        />
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
          <CardContent className="pt-6">
            <Tabs defaultValue="value">
              <div className="flex items-center justify-between mb-4">
                <TabsList>
                  <TabsTrigger value="value">Value</TabsTrigger>
                  <TabsTrigger value="performance">Performance</TabsTrigger>
                </TabsList>
              </div>
              <TabsContent value="value" className="mt-0">
                <ValueOverTime onInvestClick={() => setInvestOpen(true)} />
              </TabsContent>
              <TabsContent value="performance" className="mt-0">
                <PerformanceChart onInvestClick={() => setInvestOpen(true)} />
              </TabsContent>
            </Tabs>
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
