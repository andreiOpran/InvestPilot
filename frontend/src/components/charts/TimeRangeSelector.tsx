const RANGES = ["1D", "1W", "1M", "6M", "1Y", "YTD", "5Y"] as const;
export type TimeRange = (typeof RANGES)[number];

interface TimeRangeSelectorProps {
  value: TimeRange;
  onChange: (range: TimeRange) => void;
}

export function TimeRangeSelector({ value, onChange }: TimeRangeSelectorProps) {
  return (
    <div className="flex items-center gap-1 rounded-lg bg-muted p-1 w-fit">
      {RANGES.map((range) => (
        <button
          key={range}
          onClick={() => onChange(range)}
          className={`px-3 py-1.5 rounded-md text-sm font-medium transition-all duration-200 ${
            value === range
              ? "bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          {range}
        </button>
      ))}
    </div>
  );
}
