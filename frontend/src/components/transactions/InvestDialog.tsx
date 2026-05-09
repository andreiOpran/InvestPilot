import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { useQueryClient } from "@tanstack/react-query";

import { investSchema, type InvestFormValues } from "@/lib/schemas";
import { portfolioApi } from "@/api/portfolio";
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

interface InvestDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

export function InvestDialog({ open, onOpenChange, onSuccess }: InvestDialogProps) {
  const { user, setUser } = useAuthStore();
  const queryClient = useQueryClient();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const form = useForm<InvestFormValues>({
    resolver: zodResolver(investSchema) as any,
    defaultValues: {
      amount: 0,
    },
  });

  const onSubmit = async (data: InvestFormValues) => {
    setIsSubmitting(true);
    try {
      await portfolioApi.invest(data.amount);
      
      // Refetch user to update wallet balance since funds were moved
      const userRes = await userApi.getUser();
      setUser(userRes.data);

      toast.success("Investment added to portfolio");
      onOpenChange(false);
      form.reset();
      queryClient.invalidateQueries({ queryKey: ["portfolio-allocation"] });
      queryClient.invalidateQueries({ queryKey: ["portfolio-history"] });
      queryClient.invalidateQueries({ queryKey: ["transactions"] });
      if (onSuccess) {
        onSuccess();
      }
    } catch (error: any) {
      if (error.response?.status === 400 && (error.response?.data?.error ?? "").toLowerCase().includes("insufficient")) {
        form.setError("amount", { type: "manual", message: "Insufficient balance in wallet" });
      } else {
        toast.error("Failed to process investment. Please try again.");
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
          <DialogTitle>Invest Funds</DialogTitle>
          <DialogDescription>
            Move funds from your wallet into the InvestPilot portfolio.
          </DialogDescription>
        </DialogHeader>

        <div className="bg-muted p-4 rounded-md mb-4 text-sm flex justify-between items-center">
          <span className="text-muted-foreground">Available to Invest:</span>
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
                  <FormLabel>Investment Amount (USD)</FormLabel>
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
                label="Confirm Investment"
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
