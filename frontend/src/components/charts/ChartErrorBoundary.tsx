import React from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { AlertCircle } from "lucide-react";

interface ChartErrorBoundaryProps {
  children: React.ReactNode;
}

interface ChartErrorBoundaryState {
  hasError: boolean;
  error?: Error;
}

export class ChartErrorBoundary extends React.Component<
  ChartErrorBoundaryProps,
  ChartErrorBoundaryState
> {
  constructor(props: ChartErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): ChartErrorBoundaryState {
    return { hasError: true, error };
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex items-center justify-center h-[400px] p-8">
          <Alert variant="destructive" className="max-w-md">
            <AlertCircle />
            <AlertTitle>Chart data could not be displayed</AlertTitle>
            <AlertDescription>
              {this.state.error?.message || "An unexpected error occurred."}
            </AlertDescription>
          </Alert>
        </div>
      );
    }
    return this.props.children;
  }
}

export function ChartSkeleton() {
  return (
    <div className="space-y-3">
      <div className="flex items-end gap-1 h-[350px]">
        {Array.from({ length: 20 }).map((_, i) => (
          <Skeleton
            key={i}
            className="flex-1 rounded-sm"
            style={{ height: `${30 + Math.random() * 70}%` }}
          />
        ))}
      </div>
      <Skeleton className="h-4 w-full" />
    </div>
  );
}
