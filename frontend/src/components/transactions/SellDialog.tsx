import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { TrendingDown } from "lucide-react";

import { sellSchema, type SellFormValues } from "@/lib/schemas";
import { portfolioApi } from "@/api/portfolio";
import { userApi } from "@/api/user";
import { useAuthStore } from "@/stores/authStore";
import { formatUSDFull } from "@/lib/format";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { CurrencyInput } from "@/components/ui/CurrencyInput";
import { SwipeToConfirmButton } from "@/components/ui/SwipeToConfirmButton";

interface SellDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  portfolioValue?: number;
}

export function SellDialog({ open, onOpenChange, portfolioValue }: SellDialogProps) {
  const { setUser } = useAuthStore();
  const queryClient = useQueryClient();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const form = useForm<SellFormValues>({
    resolver: zodResolver(sellSchema) as any,
    defaultValues: { amount: 0 },
  });

  const amount = form.watch("amount");

  const onSubmit = async (data: SellFormValues) => {
    setIsSubmitting(true);
    try {
      await portfolioApi.sell(data.amount);

      // refresh wallet balance and portfolio cache
      const userRes = await userApi.getUser();
      setUser(userRes.data);
      queryClient.invalidateQueries({ queryKey: ["portfolio-allocation"] });
      queryClient.invalidateQueries({ queryKey: ["portfolio-history"] });
      queryClient.invalidateQueries({ queryKey: ["transactions"] });

      toast.success("Funds returned to wallet");
      setTimeout(() => { onOpenChange(false); form.reset(); }, 1200);
    } catch (error: any) {
      const msg = error.response?.data?.error || "Failed to process sell";
      if (
        error.response?.status === 400 &&
        msg.toLowerCase().includes("exceeds")
      ) {
        form.setError("amount", {
          type: "manual",
          message: "Amount exceeds current portfolio value",
        });
      } else if (
        error.response?.status === 400 &&
        msg.toLowerCase().includes("no active")
      ) {
        toast.error("No active portfolio to sell from");
      } else {
        toast.error(msg);
      }
      throw error;
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <TrendingDown className="h-5 w-5 text-red-500" />
            Sell Portfolio
          </DialogTitle>
          <DialogDescription>
            Liquidate a portion of your portfolio and return funds to your
            wallet. Shares are reduced proportionally across all holdings.
          </DialogDescription>
        </DialogHeader>

        {portfolioValue !== undefined && (
          <div className="bg-muted p-4 rounded-md text-sm flex justify-between items-center">
            <span className="text-muted-foreground">Available to Sell:</span>
            <span className="font-semibold text-foreground">
              {formatUSDFull(portfolioValue)}
            </span>
          </div>
        )}

        <Form {...form}>
          <form className="space-y-4">
            <FormField
              control={form.control}
              name="amount"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Sell Amount (USD)</FormLabel>
                  <FormControl>
                    <CurrencyInput
                      placeholder="Enter amount..."
                      value={field.value || 0}
                      onChange={field.onChange}
                      onBlur={field.onBlur}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="flex justify-end pt-2">
              <SwipeToConfirmButton
                label="Confirm Sell"
                onConfirm={form.handleSubmit(onSubmit)}
                isLoading={isSubmitting}
                disabled={!amount || amount <= 0 || (portfolioValue !== undefined && Math.round(amount * 100) > Math.round(portfolioValue * 100))}
                open={open}
                variant="destructive"
              />
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
