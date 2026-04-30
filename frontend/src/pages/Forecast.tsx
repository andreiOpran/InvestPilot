import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2, TrendingUp, Clock, RefreshCw } from "lucide-react";

import { useAuthStore } from "@/stores/authStore";
import { useForecast } from "@/hooks/useForecast";
import { forecastSchema } from "@/lib/schemas";
import type { ForecastFormValues } from "@/lib/schemas";
import { ConeOfUncertainty } from "@/components/charts/ConeOfUncertainty";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Slider } from "@/components/ui/slider";
import { Badge } from "@/components/ui/badge";

const riskLabels: Record<number, string> = {
  1: 'Very Conservative',
  2: 'Conservative',
  3: 'Balanced',
  4: 'Growth',
  5: 'Aggressive Growth',
};

export function Forecast() {
  const { user } = useAuthStore();
  const { status, forecastData, inputs, submitForecast, reset } = useForecast();

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

  return (
    <div className="p-8 max-w-5xl mx-auto space-y-6">
      <div className="flex justify-between items-end">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Forecast Engine</h1>
          <p className="text-muted-foreground mt-1">Project your portfolio's future value based on Monte Carlo simulations.</p>
        </div>
        {status === "complete" && (
          <Button variant="outline" onClick={reset} className="gap-2">
            <RefreshCw className="h-4 w-4" />
            New Forecast
          </Button>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left Column: Form & Context */}
        <div className="lg:col-span-1 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Your Profile</CardTitle>
              <CardDescription>Forecasts use your personalized allocation.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="rounded-xl border border-border/60 bg-muted/30 p-3 flex items-center justify-between">
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <TrendingUp className="h-4 w-4" />
                  Risk
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-sm font-semibold">{riskLabels[user?.risk_tolerance || 3]}</span>
                  <Badge variant="outline" className="text-xs">Lvl {user?.risk_tolerance || 3}</Badge>
                </div>
              </div>
              <div className="rounded-xl border border-border/60 bg-muted/30 p-3 flex items-center justify-between">
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Clock className="h-4 w-4" />
                  Horizon
                </div>
                <span className="text-sm font-semibold">{user?.investment_horizon || 10} years</span>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Parameters</CardTitle>
              <CardDescription>Adjust inputs to see different outcomes.</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
                <div className="space-y-2">
                  <Label htmlFor="initial_investment">Initial Investment ($)</Label>
                  <Input 
                    id="initial_investment" 
                    type="number" 
                    step="1"
                    disabled={status === "submitting" || status === "polling" || status === "complete"}
                    {...form.register("initial_investment")} 
                  />
                  {form.formState.errors.initial_investment && (
                    <p className="text-sm text-red-500">{form.formState.errors.initial_investment.message}</p>
                  )}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="monthly_contribution">Monthly Contribution ($)</Label>
                  <Input 
                    id="monthly_contribution" 
                    type="number" 
                    step="1"
                    disabled={status === "submitting" || status === "polling" || status === "complete"}
                    {...form.register("monthly_contribution")} 
                  />
                  {form.formState.errors.monthly_contribution && (
                    <p className="text-sm text-red-500">{form.formState.errors.monthly_contribution.message}</p>
                  )}
                </div>

                <div className="space-y-4 pt-2">
                  <div className="flex items-center justify-between">
                    <Label>Years to Forecast</Label>
                    <span className="font-mono text-sm">{form.watch("years")} yrs</span>
                  </div>
                  <Slider
                    disabled={status === "submitting" || status === "polling" || status === "complete"}
                    min={1}
                    max={50}
                    step={1}
                    value={[form.watch("years")]}
                    onValueChange={(vals) => form.setValue("years", vals[0])}
                  />
                </div>

                {status !== "complete" && (
                  <Button 
                    type="submit" 
                    className="w-full mt-2" 
                    disabled={status === "submitting" || status === "polling"}
                  >
                    {status === "submitting" || status === "polling" ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        Computing...
                      </>
                    ) : (
                      "Run Simulation"
                    )}
                  </Button>
                )}
              </form>
            </CardContent>
          </Card>
        </div>

        {/* Right Column: Chart area */}
        <div className="lg:col-span-2">
          <Card className="h-full min-h-[500px] flex flex-col">
            <CardHeader>
              <CardTitle>Cone of Uncertainty</CardTitle>
              <CardDescription>
                Range of potential future values based on 10,000 simulated market scenarios.
              </CardDescription>
            </CardHeader>
            <CardContent className="flex-1 flex flex-col items-center justify-center relative">
              {status === "idle" && (
                <div className="text-center text-muted-foreground p-8 max-w-sm">
                  <TrendingUp className="h-12 w-12 mx-auto mb-4 opacity-20" />
                  <p>Enter your parameters and run the simulation to generate a forecast.</p>
                </div>
              )}

              {(status === "submitting" || status === "polling") && (
                <div className="text-center space-y-4">
                  <Loader2 className="h-8 w-8 animate-spin text-primary mx-auto" />
                  <p className="text-muted-foreground animate-pulse">Running Monte Carlo simulation...</p>
                </div>
              )}

              {status === "complete" && forecastData && (
                <div className="w-full h-full flex flex-col">
                  <div className="flex gap-4 mb-2 px-4">
                    <div className="text-sm">
                      <span className="text-muted-foreground">Historical Volatility: </span>
                      <span className="font-mono">{(forecastData.stats.historical_annual_volatility * 100).toFixed(2)}%</span>
                    </div>
                    <div className="text-sm">
                      <span className="text-muted-foreground">Expected Return: </span>
                      <span className="font-mono">{(forecastData.stats.historical_annual_return * 100).toFixed(2)}%</span>
                    </div>
                  </div>
                  <ConeOfUncertainty data={forecastData} inputs={inputs!} />
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
