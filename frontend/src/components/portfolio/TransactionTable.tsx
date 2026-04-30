import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { format, parseISO } from "date-fns";
import {
  ArrowUpDown,
  ArrowDown,
  ArrowUp,
  Receipt,
  ChevronsLeft,
  ChevronLeft,
  ChevronRight,
  ChevronsRight,
} from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { portfolioApi } from "@/api/portfolio";

/* ------------------------------------------------------------------ */
/* Types                                                               */
/* ------------------------------------------------------------------ */

interface UnifiedTransaction {
  id: number;
  source: string;
  type: string;
  amount: number;
  status: string;
  timestamp: string;
}

interface PaginatedResponse {
  data: UnifiedTransaction[];
  total_count: number;
  page: number;
  limit: number;
}

type SortField = "timestamp" | "amount";
type SortDir = "asc" | "desc";
type FilterGroup = "all" | "funding" | "portfolio";

const ROWS_PER_PAGE = 10;

/* ------------------------------------------------------------------ */
/* Helpers                                                             */
/* ------------------------------------------------------------------ */

function formatUSD(value: number) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(value);
}

const TYPE_CONFIG: Record<
  string,
  { label: string; colorClass: string; sign: string }
> = {
  DEPOSIT: {
    label: "Deposit",
    colorClass: "bg-blue-500/15 text-blue-400 border-blue-500/20",
    sign: "+",
  },
  CASHOUT: {
    label: "Cashout",
    colorClass: "bg-orange-500/15 text-orange-400 border-orange-500/20",
    sign: "−",
  },
  INVEST: {
    label: "Invest",
    colorClass: "bg-green-500/15 text-green-400 border-green-500/20",
    sign: "−",
  },
  SELL: {
    label: "Sell",
    colorClass: "bg-red-500/15 text-red-400 border-red-500/20",
    sign: "+",
  },
  WITHDRAWAL: {
    label: "Cashout",
    colorClass: "bg-orange-500/15 text-orange-400 border-orange-500/20",
    sign: "−",
  },
};

/* ------------------------------------------------------------------ */
/* Sub-components                                                      */
/* ------------------------------------------------------------------ */

function SkeletonRows({ count = 5 }: { count?: number }) {
  return (
    <>
      {Array.from({ length: count }).map((_, i) => (
        <TableRow key={i}>
          <TableCell>
            <Skeleton className="h-4 w-24" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-16 rounded-full" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-20" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-12" />
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-16 gap-4 text-center">
      <div className="rounded-full bg-primary/10 p-4">
        <Receipt className="h-8 w-8 text-primary" />
      </div>
      <div>
        <p className="font-semibold text-foreground">No transactions yet</p>
        <p className="text-sm text-muted-foreground mt-1">
          Make your first deposit to get started
        </p>
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Main Component                                                      */
/* ------------------------------------------------------------------ */

export function TransactionTable() {
  // --- local state ---
  const [page, setPage] = useState(1);
  const [filter, setFilter] = useState<FilterGroup>("all");
  const [sortField, setSortField] = useState<SortField>("timestamp");
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  // We fetch up to 100 transactions (backend limit) so client-side filtering/sorting works.
  const { data, isLoading } = useQuery<PaginatedResponse>({
    queryKey: ["transactions"],
    queryFn: () =>
      portfolioApi.getTransactions(1, 100).then((res) => res.data),
    staleTime: 30_000,
  });

  const allRows = data?.data ?? [];

  // --- client-side filtering ---
  const filteredRows = useMemo(() => {
    if (filter === "all") return allRows;
    if (filter === "funding")
      return allRows.filter(
        (t) => t.type === "DEPOSIT" || t.type === "WITHDRAWAL" || t.type === "CASHOUT"
      );
    // portfolio
    return allRows.filter(
      (t) => t.type === "INVEST" || t.type === "SELL"
    );
  }, [allRows, filter]);

  // --- client-side sorting ---
  const sortedRows = useMemo(() => {
    const copy = [...filteredRows];
    copy.sort((a, b) => {
      let cmp: number;
      if (sortField === "timestamp") {
        cmp =
          new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime();
      } else {
        cmp = a.amount - b.amount;
      }
      return sortDir === "asc" ? cmp : -cmp;
    });
    return copy;
  }, [filteredRows, sortField, sortDir]);

  // --- pagination ---
  const totalPages = Math.max(1, Math.ceil(sortedRows.length / ROWS_PER_PAGE));
  const safePage = Math.min(page, totalPages);
  const pagedRows = sortedRows.slice(
    (safePage - 1) * ROWS_PER_PAGE,
    safePage * ROWS_PER_PAGE
  );

  // Reset page when filter changes
  const setFilterAndReset = (f: FilterGroup) => {
    setFilter(f);
    setPage(1);
  };

  const toggleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortField(field);
      setSortDir("desc");
    }
  };

  const SortIcon = ({ field }: { field: SortField }) => {
    if (sortField !== field)
      return <ArrowUpDown className="ml-1 h-3.5 w-3.5 text-muted-foreground" />;
    return sortDir === "asc" ? (
      <ArrowUp className="ml-1 h-3.5 w-3.5" />
    ) : (
      <ArrowDown className="ml-1 h-3.5 w-3.5" />
    );
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between flex-wrap gap-2">
        <div>
          <h3 className="text-lg font-semibold">Transaction History</h3>
          <p className="text-sm text-muted-foreground">
            All deposits, cashouts, investments, and sells
          </p>
        </div>

        {/* Filter tabs */}
        <div className="flex bg-muted/50 p-1 rounded-lg">
          {(
            [
              ["all", "All"],
              ["funding", "Deposits & Cashouts"],
              ["portfolio", "Investments & Sells"],
            ] as [FilterGroup, string][]
          ).map(([key, label]) => (
            <Button
              key={key}
              variant={filter === key ? "secondary" : "ghost"}
              size="sm"
              onClick={() => setFilterAndReset(key)}
              className="h-8 px-3 text-xs"
            >
              {label}
            </Button>
          ))}
        </div>
      </div>

      {/* Table */}
      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="text-left">
                <button
                  className="flex items-center cursor-pointer hover:text-foreground transition-colors"
                  onClick={() => toggleSort("timestamp")}
                >
                  Date
                  <SortIcon field="timestamp" />
                </button>
              </TableHead>
              <TableHead className="text-center">Type</TableHead>
              <TableHead className="text-right">
                <button
                  className="flex items-center justify-end w-full cursor-pointer hover:text-foreground transition-colors"
                  onClick={() => toggleSort("amount")}
                >
                  Amount
                  <SortIcon field="amount" />
                </button>
              </TableHead>
              <TableHead className="text-right">Status</TableHead>
            </TableRow>
          </TableHeader>

          <TableBody>
            {isLoading ? (
              <SkeletonRows count={5} />
            ) : pagedRows.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} className="h-auto p-0">
                  <EmptyState />
                </TableCell>
              </TableRow>
            ) : (
              pagedRows.map((tx) => {
                const cfg = TYPE_CONFIG[tx.type] ?? TYPE_CONFIG.DEPOSIT;
                return (
                  <TableRow key={`${tx.source}-${tx.id}`}>
                    <TableCell className="text-left text-muted-foreground">
                      {format(parseISO(tx.timestamp), "MMM dd, yyyy 'at' HH:mm")}
                    </TableCell>
                    <TableCell className="text-center">
                      <Badge
                        variant="outline"
                        className={cfg.colorClass}
                      >
                        {cfg.label}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right font-mono font-medium">
                      <span
                        className={
                          cfg.sign === "+"
                            ? "text-green-400"
                            : "text-red-400"
                        }
                      >
                        {cfg.sign}
                        {formatUSD(tx.amount)}
                      </span>
                    </TableCell>
                    <TableCell className="text-right">
                      <span className="text-xs text-muted-foreground capitalize">
                        {tx.status.toLowerCase()}
                      </span>
                    </TableCell>
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </div>

      {/* Pagination */}
      {!isLoading && sortedRows.length > 0 && (
        <div className="flex items-center justify-between text-sm mt-4">
          <span className="font-medium">
            Page {safePage} of {totalPages}
          </span>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              disabled={safePage <= 1}
              onClick={() => setPage(1)}
            >
              <ChevronsLeft className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              disabled={safePage <= 1}
              onClick={() => setPage((p) => p - 1)}
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              disabled={safePage >= totalPages}
              onClick={() => setPage((p) => p + 1)}
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              disabled={safePage >= totalPages}
              onClick={() => setPage(totalPages)}
            >
              <ChevronsRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
