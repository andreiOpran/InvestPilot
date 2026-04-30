import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  ReferenceLine,
} from "recharts";
import { format, parseISO } from "date-fns";
import { TrendingUp } from "lucide-react";
import { Button } from "@/components/ui/button";
import { TimeRangeSelector, type TimeRange } from "./TimeRangeSelector";
import { ChartErrorBoundary, ChartSkeleton } from "./ChartErrorBoundary";
import { portfolioApi } from "@/api/portfolio";

function formatPct(value: number) {
  const sign = value >= 0 ? "+" : "";
  return `${sign}${value.toFixed(2)}%`;
}

function formatUSD(value: number) {
  const sign = value >= 0 ? "+" : "";
  return `${sign}${new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 0,
  }).format(value)}`;
}

interface CustomTooltipProps {
  active?: boolean;
  payload?: any[];
  label?: string;
  data: PerformancePoint[];
  range: TimeRange;
}

interface PerformancePoint {
  timestamp: string;
  return_percentage: number;
  portfolio_value: number;
  net_contributions: number;
  absoluteChange: number;
}

function CustomTooltip({ active, payload, label, data, range }: CustomTooltipProps) {
  if (!active || !payload || payload.length === 0) return null;

  const returnPct: number = payload[0]?.value ?? 0;
  const isGain = returnPct >= 0;

  // find matching data point for absolute change
  const point = data.find((d) => d.timestamp === label);
  const absoluteChange = point?.absoluteChange ?? 0;

  const displayDate = label
    ? format(parseISO(label), range === "1D" ? "HH:mm, d MMM" : range === "1W" ? "HH:mm, d MMM" : "PPP")
    : "";

  return (
    <div className="rounded-lg border bg-popover p-3 shadow-lg text-sm space-y-1.5 min-w-[200px]">
      <p className="font-medium text-foreground">{displayDate}</p>
      <div className="pt-1.5" />
      <div className="flex justify-between gap-4">
        <span className="text-muted-foreground">Return (since range start)</span>
        <span
          className={`font-mono font-semibold ${isGain ? "text-emerald-400" : "text-red-400"}`}
        >
          {formatPct(returnPct)}
        </span>
      </div>
      <div className="flex justify-between gap-4">
        <span className="text-muted-foreground">Value Change</span>
        <span
          className={`font-mono ${isGain ? "text-emerald-400" : "text-red-400"}`}
        >
          {formatUSD(absoluteChange)}
        </span>
      </div>
    </div>
  );
}

// Custom dot that renders green above 0 and red below
function CustomDot(props: any) {
  const { cx, cy, value } = props;
  if (cx === undefined || cy === undefined) return null;
  const color = value >= 0 ? "#10b981" : "#ef4444";
  return <circle cx={cx} cy={cy} r={3} fill={color} stroke={color} strokeWidth={1} />;
}

interface PerformanceChartProps {
  onInvestClick?: () => void;
}

export function PerformanceChart({ onInvestClick }: PerformanceChartProps) {
  const [range, setRange] = useState<TimeRange>("1M");

  const { data, isLoading, isFetching, isError } = useQuery({
    queryKey: ["portfolio-history", range],
    queryFn: () => portfolioApi.getHistory(range).then((res) => res.data),
    staleTime: 60_000,
  });

  // Enrich data points with absoluteChange from start value
  const enriched: PerformancePoint[] =
    data?.data?.map((point: any) => {
      const startValue = data.data[0]?.portfolio_value ?? 0;
      return {
        ...point,
        absoluteChange: point.portfolio_value - startValue,
      };
    }) ?? [];

  const hasData = enriched.length > 0;

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-end flex-wrap gap-2">
        <TimeRangeSelector value={range} onChange={setRange} />
      </div>

      <ChartErrorBoundary>
        {isLoading ? (
          <ChartSkeleton />
        ) : isError ? (
          <div className="flex items-center justify-center h-[400px] text-muted-foreground">
            Failed to load chart data.
          </div>
        ) : !hasData ? (
          <EmptyPortfolioState onInvestClick={onInvestClick} />
        ) : (
          <div
            className="transition-opacity duration-300"
            style={{ opacity: isFetching ? 0.4 : 1 }}
          >
            <ResponsiveContainer width="100%" height={400}>
              <LineChart data={enriched} margin={{ top: 10, right: 10, left: 10, bottom: 0 }}>
                <defs>
                  <linearGradient id="performanceGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#10b981" stopOpacity={0.8} />
                    <stop offset="50%" stopColor="#10b981" stopOpacity={0.5} />
                    <stop offset="50%" stopColor="#ef4444" stopOpacity={0.5} />
                    <stop offset="100%" stopColor="#ef4444" stopOpacity={0.8} />
                  </linearGradient>
                </defs>

                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" strokeOpacity={0.5} />
                <XAxis
                  dataKey="timestamp"
                  tick={false}
                  axisLine={false}
                  tickLine={false}
                />
                <YAxis
                  tickFormatter={formatPct}
                  tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
                  axisLine={false}
                  tickLine={false}
                  width={65}
                />
                <Tooltip
                  content={<CustomTooltip data={enriched} range={range} />}
                  cursor={{ stroke: "var(--border)", strokeWidth: 1 }}
                />

                {/* Zero baseline */}
                <ReferenceLine
                  y={0}
                  stroke="oklch(0.705 0.015 286.067)"
                  strokeWidth={1.5}
                  strokeDasharray="4 3"
                />

                <Line
                  type="monotone"
                  dataKey="return_percentage"
                  stroke="url(#performanceGradient)"
                  strokeWidth={2}
                  dot={["6M", "1Y", "YTD", "5Y"].includes(range) ? false : <CustomDot />}
                  activeDot={{ r: 5 }}
                  name="Return (%)"
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}
      </ChartErrorBoundary>
    </div>
  );
}

function EmptyPortfolioState({ onInvestClick }: { onInvestClick?: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center h-[400px] rounded-xl border border-dashed border-border bg-muted/20 p-8 text-center gap-4">
      <div className="rounded-full bg-primary/10 p-4">
        <TrendingUp className="h-8 w-8 text-primary" />
      </div>
      <div>
        <p className="font-semibold text-foreground">No portfolio data yet</p>
        <p className="text-sm text-muted-foreground mt-1">
          Invest to start tracking your portfolio performance
        </p>
      </div>
      {onInvestClick && (
        <Button onClick={onInvestClick} variant="outline" size="sm">
          Invest Now
        </Button>
      )}
    </div>
  );
}
