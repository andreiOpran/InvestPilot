import { useMemo, useState } from "react";
import {
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Line,
  ComposedChart,
} from "recharts";
import type { ForecastData } from "@/hooks/useForecast";
import { Checkbox } from "@/components/ui/checkbox";
import { formatUSDNoFrac, formatUSDCompact } from "@/lib/format";

interface ConeOfUncertaintyProps {
  data: ForecastData;
  inputs: { initialInvestment: number; monthlyContribution: number };
}

export function ConeOfUncertainty({ data, inputs }: ConeOfUncertaintyProps) {
  const [showContribution, setShowContribution] = useState(true);

  const chartData = useMemo(() => {
    return data.years.map((year, i) => ({
      year,
      // For Recharts to fill between two values, the dataKey must map to an array of [min, max]
      band: [data.pessimistic_5th_percentile[i], data.optimistic_95th_percentile[i]],
      p5: data.pessimistic_5th_percentile[i],
      p50: data.expected_50th_percentile[i],
      p95: data.optimistic_95th_percentile[i],
      contribution: inputs.initialInvestment + (inputs.monthlyContribution * 12 * year),
    }));
  }, [data, inputs]);



  return (
    <div className="w-full mt-6 space-y-2 relative">
      <div className="w-full h-[400px]">
        <ResponsiveContainer width="100%" height="100%">
          <ComposedChart
          data={chartData}
          margin={{ top: 20, right: 20, bottom: 20, left: 20 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="var(--chart-grid)" vertical={false} />

          <XAxis
            dataKey="year"
            tickFormatter={(y) => `${y}Y`}
            stroke="var(--muted-foreground)"
            fontSize={12}
            tickLine={false}
            axisLine={false}
            dy={10}
          />

          <YAxis
            orientation="right"
            tickFormatter={formatUSDCompact}
            stroke="var(--muted-foreground)"
            fontSize={12}
            tickLine={false}
            axisLine={false}
            dx={10}
            domain={['auto', 'auto']}
          />

          <Tooltip
            content={({ active, payload, label }) => {
              if (active && payload && payload.length) {
                // Because we have multiple dataKeys (band, p50), we'll extract from the raw payload
                const row = payload[0].payload;
                return (
                  <div className="bg-popover border text-popover-foreground rounded-lg p-3 shadow-md space-y-1">
                    <p className="font-semibold mb-2">Year {label}</p>
                    <div className="flex items-center gap-2">
                      <div className="w-3 h-3 rounded-full bg-green-500/50" />
                      <span className="text-sm">Optimistic (P95):</span>
                      <span className="text-sm font-mono ml-auto">{formatUSDNoFrac(row.p95)}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <div className="w-3 h-3 rounded-full bg-[var(--chart-5)]" />
                      <span className="text-sm">Expected (P50):</span>
                      <span className="text-sm font-mono font-medium ml-auto">{formatUSDNoFrac(row.p50)}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <div className="w-3 h-3 rounded-full bg-red-500/50" />
                      <span className="text-sm">Pessimistic (P5):</span>
                      <span className="text-sm font-mono ml-auto">{formatUSDNoFrac(row.p5)}</span>
                    </div>
                    {showContribution && (
                      <div className="flex items-center gap-2">
                        <div className="w-3 h-3 rounded-full bg-purple-500" />
                        <span className="text-sm">Your Contribution:</span>
                        <span className="text-sm font-mono ml-auto">{formatUSDNoFrac(row.contribution)}</span>
                      </div>
                    )}
                  </div>
                );
              }
              return null;
            }}
          />



          {/* Uncertainty Band (P5 - P95) */}
          <Area
            type="monotone"
            dataKey="band"
            stroke="none"
            fill="var(--chart-5)"
            fillOpacity={0.3}
            activeDot={false}
          />

          {/* Expected Median (P50) */}
          <Line
            type="monotone"
            dataKey="p50"
            stroke="var(--chart-5)"
            strokeWidth={3}
            dot={false}
            activeDot={{ r: 6, fill: "var(--chart-5)", strokeWidth: 0 }}
          />

          {/* User Contribution */}
          {showContribution && (
            <Line
              type="monotone"
              dataKey="contribution"
              stroke="#a855f7"
              strokeWidth={2}
              strokeDasharray="5 5"
              dot={false}
              activeDot={false}
            />
          )}
        </ComposedChart>
      </ResponsiveContainer>
      </div>
      <div className="pl-4 bottom-0 left-6 flex items-center gap-2">
        <Checkbox id="show-contrib" checked={showContribution} onCheckedChange={(c) => setShowContribution(!!c)} />
        <label htmlFor="show-contrib" className="text-xs text-muted-foreground cursor-pointer select-none">
          Show Your Contribution
        </label>
      </div>
    </div>
  );
}
