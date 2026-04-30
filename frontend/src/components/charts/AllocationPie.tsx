import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  PieChart,
  Pie,
  Cell,
  Tooltip,
  ResponsiveContainer,
  Legend,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
} from "recharts";
import { PieChart as PieChartIcon, BarChart3 as BarChartIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ChartErrorBoundary, ChartSkeleton } from "./ChartErrorBoundary";
import { portfolioApi } from "@/api/portfolio";

function formatUSD(value: number) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(value);
}

const COLORS = [
  "var(--chart-1)",
  "var(--chart-2)",
  "var(--chart-3)",
  "var(--chart-4)",
  "var(--chart-5)",
];

const USD_COLOR = "var(--muted)"; // neutral muted color from shadcn theme

interface CustomTooltipProps {
  active?: boolean;
  payload?: any[];
  totalValue: number;
}

function CustomTooltip({ active, payload, totalValue }: CustomTooltipProps) {
  if (!active || !payload || payload.length === 0) return null;

  const data = payload[0].payload;
  const pct = totalValue > 0 ? (data.value / totalValue) * 100 : 0;

  return (
    <div className="rounded-lg border bg-popover p-3 shadow-lg text-sm space-y-1.5 min-w-[150px]">
      <div className="flex items-center gap-2">
        <div
          className="w-3 h-3 rounded-full"
          style={{ backgroundColor: data.fill }}
        />
        <span className="font-semibold">{data.name}</span>
      </div>
      <div className="flex justify-between gap-4">
        <span className="text-muted-foreground">Allocation</span>
        <span className="font-mono font-medium">{pct.toFixed(1)}%</span>
      </div>
      <div className="flex justify-between gap-4">
        <span className="text-muted-foreground">Value</span>
        <span className="font-mono font-medium">{formatUSD(data.value)}</span>
      </div>
    </div>
  );
}

interface AllocationPieProps {
  showTitle?: boolean;
}

export function AllocationPie({ showTitle = true }: AllocationPieProps) {
  const [chartType, setChartType] = useState<"pie" | "bar">("pie");

  const { data, isLoading, isError } = useQuery({
    queryKey: ["portfolio-allocation"],
    queryFn: () => portfolioApi.getPortfolio().then((res) => res.data),
    staleTime: 60_000,
  });

  const holdings = data?.holdings || [];
  const hasData = holdings.length > 0;
  const totalValue = data?.live_total_value || 0;

  // Group holdings by ticker to handle potential multiple entries
  const groupedHoldings = holdings.reduce((acc: any, h: any) => {
    if (!acc[h.ticker]) {
      acc[h.ticker] = { name: h.ticker, value: 0 };
    }
    acc[h.ticker].value += h.current_value;
    return acc;
  }, {});

  const pieData = Object.values(groupedHoldings).map((h: any, index: number) => ({
    name: h.name,
    value: h.value,
    fill: h.name === "USD" ? USD_COLOR : COLORS[index % COLORS.length],
  }));

  // Sort so USD is last visually
  pieData.sort((a: any, b: any) => {
    if (a.name === "USD") return 1;
    if (b.name === "USD") return -1;
    return b.value - a.value;
  });

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        {showTitle ? (
          <div>
            <h3 className="text-lg font-semibold">Asset Allocation</h3>
            <p className="text-sm text-muted-foreground">
              Current distribution of your active portfolio
            </p>
          </div>
        ) : <div />}
        {hasData && (
          <div className="flex bg-muted/50 p-1 rounded-lg">
            <Button
              variant={chartType === "pie" ? "secondary" : "ghost"}
              size="sm"
              onClick={() => setChartType("pie")}
              className="h-8 px-3"
            >
              <PieChartIcon className="h-4 w-4 mr-2" />
              Pie
            </Button>
            <Button
              variant={chartType === "bar" ? "secondary" : "ghost"}
              size="sm"
              onClick={() => setChartType("bar")}
              className="h-8 px-3"
            >
              <BarChartIcon className="h-4 w-4 mr-2" />
              Bar
            </Button>
          </div>
        )}
      </div>

      <ChartErrorBoundary>
        {isLoading ? (
          <ChartSkeleton />
        ) : isError ? (
          <div className="flex items-center justify-center h-[400px] text-muted-foreground">
            Failed to load allocation data.
          </div>
        ) : !hasData ? (
          <EmptyAllocationState />
        ) : chartType === "pie" ? (
          <ResponsiveContainer width="100%" height={400}>
            <PieChart>
              <Pie
                data={pieData}
                cx="50%"
                cy="45%"
                innerRadius={80}
                outerRadius={120}
                paddingAngle={2}
                dataKey="value"
                stroke="none"
              >
                {pieData.map((entry: any, index: number) => (
                  <Cell key={`cell-${index}`} fill={entry.fill} />
                ))}
              </Pie>
              <Tooltip content={<CustomTooltip totalValue={totalValue} />} cursor={false} />
              <Legend
                verticalAlign="bottom"
                height={36}
                formatter={(value, entry: any) => {
                  const pct = totalValue > 0 ? (entry.payload.value / totalValue) * 100 : 0;
                  return (
                    <span className="text-foreground font-medium ml-1">
                      {value} <span className="text-muted-foreground font-normal">({pct.toFixed(1)}%)</span>
                    </span>
                  );
                }}
              />
            </PieChart>
          </ResponsiveContainer>
        ) : (
          <ResponsiveContainer width="100%" height={400}>
            <BarChart data={pieData} margin={{ top: 20, right: 30, left: 20, bottom: 20 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
              <XAxis 
                dataKey="name" 
                stroke="var(--muted-foreground)" 
                fontSize={12} 
                tickLine={false} 
                axisLine={false} 
                dy={10}
              />
              <YAxis 
                stroke="var(--muted-foreground)" 
                fontSize={12} 
                tickLine={false} 
                axisLine={false}
                tickFormatter={(val) => `$${val}`}
                dx={-10}
              />
              <Tooltip content={<CustomTooltip totalValue={totalValue} />} cursor={{ fill: 'var(--muted)', opacity: 0.4 }} />
              <Bar 
                dataKey="value" 
                fill="var(--primary)" 
                radius={[4, 4, 0, 0]}
                maxBarSize={60}
              >
                {pieData.map((entry: any, index: number) => (
                  <Cell key={`cell-${index}`} fill={entry.fill} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        )}
      </ChartErrorBoundary>
    </div>
  );
}

function EmptyAllocationState() {
  return (
    <div className="flex flex-col items-center justify-center h-[400px] rounded-xl border border-dashed border-border bg-muted/20 p-8 text-center gap-4">
      <div className="rounded-full bg-primary/10 p-4">
        <PieChartIcon className="h-8 w-8 text-primary" />
      </div>
      <div>
        <p className="font-semibold text-foreground">No active investments</p>
        <p className="text-sm text-muted-foreground mt-1">
          Invest funds to see your asset allocation
        </p>
      </div>
    </div>
  );
}
