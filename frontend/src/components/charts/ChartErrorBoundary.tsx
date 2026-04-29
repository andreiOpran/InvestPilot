import React from "react";
import { Skeleton } from "@/components/ui/skeleton";

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
        <div className="flex flex-col items-center justify-center h-[400px] rounded-xl border border-destructive/30 bg-destructive/5 p-8 text-center gap-3">
          <p className="text-destructive font-medium">Failed to render chart</p>
          <p className="text-muted-foreground text-sm">
            {this.state.error?.message || "An unexpected error occurred."}
          </p>
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
