import { useState, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { forecastApi } from "@/api/forecast";

export type ForecastStatus = "idle" | "submitting" | "polling" | "complete" | "error";

export interface ForecastData {
  years: number[];
  pessimistic_5th_percentile: number[];
  expected_50th_percentile: number[];
  optimistic_95th_percentile: number[];
  stats: {
    historical_annual_return: number;
    historical_annual_volatility: number;
  };
}

export function useForecast() {
  const [status, setStatus] = useState<ForecastStatus>("idle");
  const [taskId, setTaskId] = useState<string | null>(null);
  const [forecastData, setForecastData] = useState<ForecastData | null>(null);
  const [inputs, setInputs] = useState<{initialInvestment: number, monthlyContribution: number} | null>(null);

  const { data: statusData, error: pollError } = useQuery({
    queryKey: ["forecastStatus", taskId],
    queryFn: () => forecastApi.getForecastStatus(taskId!),
    enabled: !!taskId && status === "polling",
    refetchInterval: 2000,
  });

  useEffect(() => {
    if (pollError) {
      setStatus("error");
      toast.error("Forecast computation failed");
    } else if (statusData?.data) {
      const { status: taskStatus, payload } = statusData.data;

      if (taskStatus === "complete") {
        setForecastData(payload as ForecastData);
        setStatus("complete");
      } else if (taskStatus === "error" || taskStatus === "failed") {
        setStatus("error");
        toast.error("Forecast computation failed");
      }
    }
  }, [statusData, pollError]);

  const submitForecast = async (initialInvestment: number, monthlyContribution: number, years: number) => {
    try {
      setStatus("submitting");
      setInputs({ initialInvestment, monthlyContribution });
      const res = await forecastApi.requestForecast(initialInvestment, monthlyContribution, years);
      setTaskId(res.data.task_id);
      setStatus("polling");
    } catch (err) {
      setStatus("error");
      toast.error("Failed to submit forecast request");
    }
  };

  const reset = () => {
    setStatus("idle");
    setTaskId(null);
    setForecastData(null);
    setInputs(null);
  };

  return {
    status,
    forecastData,
    inputs,
    submitForecast,
    reset,
  };
}
