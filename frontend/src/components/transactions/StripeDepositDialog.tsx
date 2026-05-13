import { useState, useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";
import { formatUSDFull } from "@/lib/format";
import { loadStripe } from "@stripe/stripe-js";
import {
  Elements,
  PaymentElement,
  useStripe,
  useElements,
} from "@stripe/react-stripe-js";

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
import { CurrencyInput } from "@/components/ui/CurrencyInput";
import { Button } from "@/components/ui/button";
import { SwipeToConfirmButton } from "@/components/ui/SwipeToConfirmButton";

// Make sure to call `loadStripe` outside of a component’s render to avoid
// recreating the `Stripe` object on every render.
const PUBLISHABLE_KEY = import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY;
if (!PUBLISHABLE_KEY) {
  console.warn("Stripe Publishable Key is missing from environment variables.");
}
const stripePromise = loadStripe(PUBLISHABLE_KEY || "", {
  developerTools: { assistant: { enabled: false } },
});

const stripeAppearance = {
  theme: 'night' as const,
  variables: {
    colorPrimary: '#4ade80',
    colorBackground: '#27272f',
    colorText: '#fafafa',
    colorDanger: '#f87171',
    colorTextPlaceholder: '#71717a',
    colorTextSecondary: '#a1a1aa',
    fontFamily: "system-ui, 'Segoe UI', Roboto, sans-serif",
    borderRadius: '7px',
    spacingUnit: '4px',
  },
  rules: {
    '.Input': {
      border: '1px solid rgba(255, 255, 255, 0.1)',
      backgroundColor: '#27272f',
    },
    '.Input:focus': {
      borderColor: '#4ade80',
      boxShadow: '0 0 0 1px #4ade80, 0 0 0 3px rgba(74, 222, 128, 0.12)',
    },
    '.Input--invalid': {
      borderColor: '#f87171',
      boxShadow: '0 0 0 1px #f87171',
    },
    '.Tab': {
      border: '1px solid rgba(255, 255, 255, 0.1)',
      backgroundColor: '#27272f',
    },
    '.Tab:hover': {
      border: '1px solid rgba(255, 255, 255, 0.2)',
      color: '#fafafa',
    },
    '.Tab--selected': {
      border: '1px solid #4ade80',
      boxShadow: '0 0 0 1px #4ade80',
    },
    '.Label': {
      color: '#a1a1aa',
    },
    '.Error': {
      color: '#f87171',
    },
  },
};

interface StripeDepositDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function StripeDepositDialog({ open, onOpenChange }: StripeDepositDialogProps) {
  const [clientSecret, setClientSecret] = useState<string | null>(null);
  const [amount, setAmount] = useState<number | null>(null);

  // Reset state when dialog is closed
  useEffect(() => {
    if (!open) {
      setTimeout(() => {
        setClientSecret(null);
        setAmount(null);
      }, 300); // Wait for closing animation
    }
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Deposit via Stripe</DialogTitle>
          <DialogDescription>
            {clientSecret
              ? "Complete your payment below."
              : "Enter the amount you wish to deposit securely via Stripe."}
          </DialogDescription>
        </DialogHeader>

        {!clientSecret ? (
          <AmountForm
            onIntentCreated={(secret, amt) => {
              setClientSecret(secret);
              setAmount(amt);
            }}
          />
        ) : (
          <Elements stripe={stripePromise} options={{ clientSecret, appearance: stripeAppearance }}>
            <CheckoutForm
              amount={amount!}
              onSuccess={() => {
                onOpenChange(false);
              }}
            />
          </Elements>
        )}
      </DialogContent>
    </Dialog>
  );
}

// ─── STEP 1: Amount Form ──────────────────────────────────────────────

interface AmountFormProps {
  onIntentCreated: (clientSecret: string, amount: number) => void;
}

function AmountForm({ onIntentCreated }: AmountFormProps) {
  const [isSubmitting, setIsSubmitting] = useState(false);

  const form = useForm<DepositFormValues>({
    resolver: zodResolver(depositSchema) as any,
    defaultValues: {
      amount: 0,
    },
  });

  const amount = form.watch("amount");

  const onSubmit = async (data: DepositFormValues) => {
    setIsSubmitting(true);
    try {
      const response = await userApi.createDepositIntent(data.amount);
      if (response.data.client_secret) {
        onIntentCreated(response.data.client_secret, data.amount);
      } else {
        toast.error("Failed to initialize payment.");
        throw new Error("No client secret");
      }
    } catch (error: any) {
      toast.error("Failed to initialize payment. Please try again.");
      throw error;
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 pt-4">
        <FormField
          control={form.control}
          name="amount"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Amount (USD)</FormLabel>
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
          <Button type="submit" disabled={isSubmitting || !amount || amount <= 0} className="w-full">
            {isSubmitting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Initializing...
              </>
            ) : (
              "Continue to Payment"
            )}
          </Button>
        </div>
      </form>
    </Form>
  );
}

// ─── STEP 2: Checkout Form ────────────────────────────────────────────

interface CheckoutFormProps {
  amount: number;
  onSuccess: () => void;
}

function CheckoutForm({ amount, onSuccess }: CheckoutFormProps) {
  const stripe = useStripe();
  const elements = useElements();
  const { setUser } = useAuthStore();

  const [isLoading, setIsLoading] = useState(false);

  const handlePay = async () => {
    if (!stripe || !elements) return;

    setIsLoading(true);
    try {
      const { error, paymentIntent } = await stripe.confirmPayment({
        elements,
        confirmParams: {},
        redirect: "if_required",
      });

      if (error) {
        toast.error(error.message ?? "An unexpected error occurred.");
        throw error;
      } else if (paymentIntent && paymentIntent.status === "succeeded") {
        toast.success("Deposit submitted! Funds will arrive after confirmation.");
        try {
          const userRes = await userApi.getUser();
          setUser(userRes.data);
        } catch (e) {
          console.error("Failed to refresh user data after stripe success", e);
        }
        setTimeout(() => onSuccess(), 2200);
      } else if (paymentIntent && paymentIntent.status === "processing") {
        toast.info("Payment is processing. We'll update you when it succeeds.");
        setTimeout(() => onSuccess(), 2200);
      } else {
        toast.error("Payment failed. Please try again.");
        throw new Error("Payment failed");
      }
    } finally {
      setIsLoading(false);
    }
  };

  if (!stripe || !elements) {
    return (
      <div className="flex flex-col items-center justify-center p-12 space-y-4">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        <p className="text-sm text-muted-foreground">Loading payment security...</p>
      </div>
    );
  }

  return (
    <div className="space-y-4 pt-4">
      <PaymentElement id="payment-element" />
      <div className="flex justify-end pt-4">
        <SwipeToConfirmButton
          label={`Pay ${formatUSDFull(amount)}`}
          onConfirm={handlePay}
          isLoading={isLoading}
        />
      </div>
    </div>
  );
}
