import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";

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
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";

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

  const onSubmit = async (data: CashoutFormValues) => {
    setIsSubmitting(true);
    try {
      await userApi.cashout(data.amount);
      
      // Refetch user to update wallet balance
      const userRes = await userApi.getUser();
      setUser(userRes.data);

      toast.success("Withdrawal processed successfully!");
      onOpenChange(false);
      form.reset();
    } catch (error: any) {
      const msg = error.response?.data?.error || "Failed to process withdrawal";
      
      // Handle the 400 insufficient balance specifically if needed
      if (error.response?.status === 400 && msg.toLowerCase().includes("insufficient")) {
        form.setError("amount", { type: "manual", message: "Insufficient balance" });
      } else {
        toast.error(msg);
      }
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
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="amount"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Amount to Withdraw (USD)</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      step="0.01"
                      placeholder="Enter amount..."
                      {...field}
                      value={field.value || ""}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="flex justify-end pt-4">
              <Button type="submit" disabled={isSubmitting} className="w-full">
                {isSubmitting ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Processing...
                  </>
                ) : (
                  "Confirm Withdrawal"
                )}
              </Button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
