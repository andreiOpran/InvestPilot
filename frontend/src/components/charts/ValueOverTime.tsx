import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  AreaChart,
  Area,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from "recharts";
import { format, parseISO } from "date-fns";
import { TrendingUp } from "lucide-react";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { TimeRangeSelector, type TimeRange } from "./TimeRangeSelector";
import { ChartErrorBoundary, ChartSkeleton } from "./ChartErrorBoundary";
import { portfolioApi } from "@/api/portfolio";

function formatUSD(value: number) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value);
}

interface CustomTooltipProps {
  active?: boolean;
  payload?: any[];
  label?: string;
  showNetContributions: boolean;
  range: TimeRange;
}

function CustomTooltip({ active, payload, label, showNetContributions, range }: CustomTooltipProps) {
  if (!active || !payload || payload.length === 0) return null;

  const portfolioValue = payload.find((p) => p.dataKey === "portfolio_value")?.value ?? 0;
  const netContributions = payload.find((p) => p.dataKey === "net_contributions")?.value ?? 0;
  const gainLoss = portfolioValue - netContributions;
  const gainLossPct = netContributions > 0 ? (gainLoss / netContributions) * 100 : 0;
  const isGain = gainLoss >= 0;

  const displayDate = label
    ? format(parseISO(label), range === "1D" ? "HH:mm, d MMM" : range === "1W" ? "HH:mm, d MMM" : "PPP")
    : "";

  return (
    <div className="rounded-lg border bg-popover p-3 shadow-lg text-sm space-y-1.5 min-w-[200px]">
      <p className="font-medium text-foreground">{displayDate}</p>
      <div className=" pt-1.5" />
      <div className="flex justify-between gap-4">
        <span className="text-muted-foreground">Portfolio Value</span>
        <span className="font-mono font-semibold">{formatUSD(portfolioValue)}</span>
      </div>
      {showNetContributions && (
        <div className="flex justify-between gap-4">
          <span className="text-muted-foreground">Net Contributions</span>
          <span className="font-mono">{formatUSD(netContributions)}</span>
        </div>
      )}
      <div className="border-t pt-1.5 flex justify-between gap-4">
        <span className="text-muted-foreground">Unrealized G/L</span>
        <span className={`font-mono font-semibold ${isGain ? "text-emerald-400" : "text-red-400"}`}>
          {isGain ? "+" : ""}{formatUSD(gainLoss)} ({isGain ? "+" : ""}{gainLossPct.toFixed(2)}%)
        </span>
      </div>
    </div>
  );
}

interface ValueOverTimeProps {
  onInvestClick?: () => void;
}

export function ValueOverTime({ onInvestClick }: ValueOverTimeProps) {
  const [range, setRange] = useState<TimeRange>("1M");
  const [showNetContributions, setShowNetContributions] = useState(true);

  const { data, isLoading, isFetching, isError } = useQuery({
    queryKey: ["portfolio-history", range],
    queryFn: () => portfolioApi.getHistory(range).then((res) => res.data),
    staleTime: 60_000,
  });

  const hasData = data?.data && data.data.length > 0;

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
              <AreaChart data={data!.data} margin={{ top: 10, right: 10, left: 20, bottom: 0 }}>
                <defs>
                  <linearGradient id="portfolioGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="var(--color-primary, #10b981)" stopOpacity={0.4} />
                    <stop offset="95%" stopColor="var(--color-primary, #10b981)" stopOpacity={0.02} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--muted-foreground)" strokeOpacity={0.2} />
                <XAxis
                  dataKey="timestamp"
                  tick={false}
                  axisLine={false}
                  tickLine={false}
                />
                <YAxis
                  tickFormatter={formatUSD}
                  tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
                  axisLine={false}
                  tickLine={false}
                  width={75}
                />
                <Tooltip
                  content={<CustomTooltip showNetContributions={showNetContributions} range={range} />}
                  cursor={{ stroke: "var(--muted-foreground)", strokeWidth: 1, strokeOpacity: 0.4 }}
                />
                <Legend
                  formatter={(value) =>
                    value === "portfolio_value"
                      ? "Portfolio Value"
                      : value === "net_contributions"
                      ? "Net Contributions"
                      : "Unrealized Gain / Loss"
                  }
                  wrapperStyle={{ fontSize: 12 }}
                />
                <Area
                  type="monotone"
                  dataKey="portfolio_value"
                  name="portfolio_value"
                  stroke="oklch(0.627 0.194 149.214)"
                  strokeWidth={2}
                  fill="url(#portfolioGradient)"
                  dot={false}
                  activeDot={{ r: 4 }}
                />
                {showNetContributions && (
                  <Line
                    type="monotone"
                    dataKey="net_contributions"
                    name="net_contributions"
                    stroke="oklch(0.705 0.015 286.067)"
                    strokeWidth={1.5}
                    strokeDasharray="5 4"
                    dot={false}
                    activeDot={{ r: 3 }}
                  />
                )}
              </AreaChart>
            </ResponsiveContainer>

            <div className="flex items-center gap-2 mt-3 pl-1">
              <Checkbox
                id="show-net-contributions"
                checked={showNetContributions}
                onCheckedChange={(checked) => setShowNetContributions(checked === true)}
              />
              <Label htmlFor="show-net-contributions" className="text-sm cursor-pointer">
                Show Net Contributions
              </Label>
            </div>
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
        <p className="font-semibold text-foreground">No portfolio data</p>
        <p className="text-sm text-muted-foreground mt-1">
          Invest funds to see your portfolio performance
        </p>
      </div>
      {/* {onInvestClick && (
        <Button onClick={onInvestClick} variant="outline" size="sm">
          Invest Now
        </Button>
      )} */}
    </div>
  );
}
