import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";

import { depositSchema, type DepositFormValues } from "@/lib/schemas";
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
import { SwipeToConfirmButton } from "@/components/ui/SwipeToConfirmButton";

interface DepositDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function DepositDialog({ open, onOpenChange }: DepositDialogProps) {
  const { setUser } = useAuthStore();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const form = useForm<DepositFormValues>({
    resolver: zodResolver(depositSchema) as any,
    defaultValues: {
      amount: 0,
    },
  });

  const onSubmit = async (data: DepositFormValues) => {
    setIsSubmitting(true);
    try {
      await userApi.deposit(data.amount);
      
      // Refetch user to update wallet balance
      const userRes = await userApi.getUser();
      setUser(userRes.data);

      toast.success("Funds added successfully!");
      onOpenChange(false);
      form.reset();
    } catch (error: any) {
      toast.error("Failed to process deposit. Please try again.");
      throw error;
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Paper Trading Deposit</DialogTitle>
          <DialogDescription>
            Add simulated funds to your paper trading wallet.
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 pt-4">
            <FormField
              control={form.control}
              name="amount"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Amount (USD)</FormLabel>
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
              <SwipeToConfirmButton
                label="Deposit Funds"
                onConfirm={form.handleSubmit(onSubmit)}
                isLoading={isSubmitting}
                open={open}
              />
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
