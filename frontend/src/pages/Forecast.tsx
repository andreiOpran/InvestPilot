import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2, TrendingUp, Clock } from "lucide-react";

import { useAuthStore } from "@/stores/authStore";
import { useForecast } from "@/hooks/useForecast";
import { forecastSchema } from "@/lib/schemas";
import type { ForecastFormValues } from "@/lib/schemas";
import { ConeOfUncertainty } from "@/components/charts/ConeOfUncertainty";
import { formatPctPlain, formatUSDNoFrac } from "@/lib/format";
import { ChartErrorBoundary } from "@/components/charts/ChartErrorBoundary";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Slider } from "@/components/ui/slider";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";

const riskLabels: Record<number, string> = {
  1: "Very Conservative",
  2: "Conservative",
  3: "Balanced",
  4: "Growth",
  5: "Aggressive Growth",
};

export function Forecast() {
  const { user } = useAuthStore();
  const { status, forecastData, inputs, submitForecast } = useForecast();

  const form = useForm<ForecastFormValues>({
    resolver: zodResolver(forecastSchema) as any,
    defaultValues: {
      initial_investment: user?.wallet_balance ? Math.floor(user.wallet_balance) : 10000,
      monthly_contribution: 500,
      years: 10,
    },
  });

  const onSubmit = (data: ForecastFormValues) => {
    submitForecast(data.initial_investment, data.monthly_contribution || 0, data.years);
  };

  const isDisabled = status === "submitting" || status === "polling";

  return (
    <div className="p-4 md:p-8 space-y-6 max-w-7xl mx-auto">

      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="space-y-0.5">
          <h1 className="text-xl font-semibold tracking-tight">Forecast Engine</h1>
          <p className="text-sm text-muted-foreground">
            Project your portfolio across 10,000 Monte Carlo simulations.
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">

        {/* Left: inputs */}
        <div className="space-y-4">

          {/* Profile summary */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-semibold tracking-tight">Your Profile</CardTitle>
              <CardDescription className="text-xs">Forecasts use your personalized allocation.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2">
              <div className="flex items-center justify-between rounded-lg border bg-muted/30 px-3 py-2.5">
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <TrendingUp className="h-3.5 w-3.5" />
                  Risk level
                </div>
                <div className="flex items-center gap-1.5">
                  <span className="text-xs font-semibold">{riskLabels[user?.risk_tolerance || 3]}</span>
                  <Badge variant="outline" className="text-[10px] h-4 px-1">
                    {user?.risk_tolerance || 3}/5
                  </Badge>
                </div>
              </div>
              <div className="flex items-center justify-between rounded-lg border bg-muted/30 px-3 py-2.5">
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <Clock className="h-3.5 w-3.5" />
                  Horizon
                </div>
                <span className="text-xs font-semibold">{user?.investment_horizon || 10} years</span>
              </div>
            </CardContent>
          </Card>

          {/* Parameters */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-semibold tracking-tight">Parameters</CardTitle>
              <CardDescription className="text-xs">Adjust inputs to model different scenarios.</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">

                <div className="space-y-1.5">
                  <Label htmlFor="initial_investment" className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                    Initial investment ($)
                  </Label>
                  <Input
                    id="initial_investment"
                    type="number"
                    step="1"
                    disabled={isDisabled}
                    {...form.register("initial_investment")}
                    className="h-9 text-sm"
                  />
                  {form.formState.errors.initial_investment && (
                    <p className="text-xs text-destructive">{form.formState.errors.initial_investment.message}</p>
                  )}
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="monthly_contribution" className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                    Monthly contribution ($)
                  </Label>
                  <Input
                    id="monthly_contribution"
                    type="number"
                    step="1"
                    disabled={isDisabled}
                    {...form.register("monthly_contribution")}
                    className="h-9 text-sm"
                  />
                  {form.formState.errors.monthly_contribution && (
                    <p className="text-xs text-destructive">{form.formState.errors.monthly_contribution.message}</p>
                  )}
                </div>

                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <Label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Years to forecast</Label>
                    <span className="font-mono text-xs font-semibold">{form.watch("years")} yrs</span>
                  </div>
                  <Slider
                    disabled={isDisabled}
                    min={1}
                    max={50}
                    step={1}
                    value={[form.watch("years")]}
                    onValueChange={(vals) => form.setValue("years", vals[0])}
                  />
                </div>

                <Separator />

                <Button
                  type="submit"
                  className="w-full h-9 font-medium text-sm"
                  disabled={status === "submitting" || status === "polling"}
                >
                  {status === "submitting" || status === "polling" ? (
                    <>
                      <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" />
                      Computing...
                    </>
                  ) : (
                    "Run simulation"
                  )}
                </Button>
              </form>
            </CardContent>
          </Card>
        </div>

        {/* Right: chart */}
        <div className="lg:col-span-2">
          <Card className="h-full min-h-[480px] flex flex-col">
            <CardHeader className="pb-3">
              <div className="flex items-start justify-between">
                <div>
                  <CardTitle className="text-sm font-semibold tracking-tight">Cone of Uncertainty</CardTitle>
                  <CardDescription className="text-xs mt-0.5">
                    Range of potential values across 10,000 simulated market scenarios.
                  </CardDescription>
                </div>
                {status === "complete" && forecastData && inputs && (
                  <div className="flex flex-col items-end gap-2 shrink-0">
                    <div className="flex gap-4 text-xs">
                      <div>
                        <span className="text-muted-foreground">Volatility: </span>
                        <span className="font-mono font-medium">
                          {formatPctPlain(forecastData.stats.historical_annual_volatility * 100)}
                        </span>
                      </div>
                      <div>
                        <span className="text-muted-foreground">Return: </span>
                        <span className="font-mono font-medium">
                          {formatPctPlain(forecastData.stats.historical_annual_return * 100)}
                        </span>
                      </div>
                    </div>
                    <div className="flex items-center gap-0 text-[11px] rounded-md border bg-muted/40 overflow-hidden">
                      <span className="px-2.5 py-1 text-muted-foreground border-r">
                        <span className="text-foreground font-mono font-medium">{formatUSDNoFrac(inputs.initialInvestment)}</span>
                        {" "}initial
                      </span>
                      <span className="px-2.5 py-1 text-muted-foreground border-r">
                        <span className="text-foreground font-mono font-medium">{formatUSDNoFrac(inputs.monthlyContribution)}</span>
                        {" "}/mo
                      </span>
                      <span className="px-2.5 py-1 text-muted-foreground">
                        <span className="text-foreground font-mono font-medium">{forecastData.years[forecastData.years.length - 1]}</span>
                        {" "}yrs
                      </span>
                    </div>
                  </div>
                )}
              </div>
            </CardHeader>
            <Separator />
            <CardContent className="flex-1 flex flex-col items-center justify-center pt-6">

              {(status === "idle" || status === "error") && (
                <div className="text-center space-y-3 max-w-xs">
                  <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted mx-auto">
                    <TrendingUp className="h-6 w-6 text-muted-foreground/40" />
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm font-medium">No simulation yet</p>
                    <p className="text-xs text-muted-foreground">
                      Enter your parameters and run the simulation to generate a forecast.
                    </p>
                  </div>
                </div>
              )}

              {(status === "submitting" || status === "polling") && (
                <div className="text-center space-y-3">
                  <Loader2 className="h-8 w-8 animate-spin text-primary mx-auto" />
                  <p className="text-sm text-muted-foreground">Running Monte Carlo simulation...</p>
                </div>
              )}

              {status === "complete" && forecastData && (
                <div className="w-full h-full">
                  <ChartErrorBoundary><ConeOfUncertainty data={forecastData} inputs={inputs!} /></ChartErrorBoundary>
                </div>
              )}

            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
