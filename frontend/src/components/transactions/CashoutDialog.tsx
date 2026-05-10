import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";

import { cashoutSchema, type CashoutFormValues } from "@/lib/schemas";
import { userApi } from "@/api/user";
import { useAuthStore } from "@/stores/authStore";

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

interface CashoutDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CashoutDialog({ open, onOpenChange }: CashoutDialogProps) {
  const { user, setUser } = useAuthStore();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const form = useForm<CashoutFormValues>({
    resolver: zodResolver(cashoutSchema) as any,
    defaultValues: {
      amount: 0,
    },
  });

  const amount = form.watch("amount");

  const onSubmit = async (data: CashoutFormValues) => {
    setIsSubmitting(true);
    try {
      await userApi.cashout(data.amount);
      
      // Refetch user to update wallet balance
      const userRes = await userApi.getUser();
      setUser(userRes.data);

      toast.success("Withdrawal processed successfully!");
      setTimeout(() => { onOpenChange(false); form.reset(); }, 1200);
    } catch (error: any) {
      const msg = error.response?.data?.error || "Failed to process withdrawal";
      if (error.response?.status === 400 && msg.toLowerCase().includes("insufficient")) {
        form.setError("amount", { type: "manual", message: "Insufficient balance" });
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
          <DialogTitle>Withdraw Funds</DialogTitle>
          <DialogDescription>
            Withdraw simulated funds from your paper trading wallet.
          </DialogDescription>
        </DialogHeader>

        <div className="bg-muted p-4 rounded-md mb-4 text-sm flex justify-between items-center">
          <span className="text-muted-foreground">Available Balance:</span>
          <span className="font-semibold text-foreground">
            ${user?.wallet_balance?.toFixed(2) || "0.00"}
          </span>
        </div>

        <Form {...form}>
          <form className="space-y-4">
            <FormField
              control={form.control}
              name="amount"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Amount to Withdraw (USD)</FormLabel>
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

            <div className="flex justify-end pt-4">
              <SwipeToConfirmButton
                label="Confirm Withdrawal"
                onConfirm={form.handleSubmit(onSubmit)}
                isLoading={isSubmitting}
                disabled={!amount || amount <= 0 || Math.round(amount * 100) > Math.round((user?.wallet_balance ?? 0) * 100)}
                open={open}
              />
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
