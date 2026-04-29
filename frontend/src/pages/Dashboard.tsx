import { useState } from "react";
import { useAuthStore } from "@/stores/authStore";
import { Button } from "@/components/ui/button";
import { LogoutButton } from "@/components/auth/LogoutButton";
import { DepositDialog } from "@/components/transactions/DepositDialog";
import { StripeDepositDialog } from "@/components/transactions/StripeDepositDialog";

export function Dashboard() {
  const { user } = useAuthStore();
  const [paperDepositOpen, setPaperDepositOpen] = useState(false);
  const [stripeDepositOpen, setStripeDepositOpen] = useState(false);

  return (
    <div className="p-8 space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <LogoutButton />
      </div>

      <div className="p-6 border rounded-xl bg-card">
        <h2 className="text-xl font-semibold mb-4">Wallet & Deposits</h2>
        <p className="text-muted-foreground mb-4">
          Current Balance: <span className="font-bold text-foreground">${user?.wallet_balance?.toFixed(2)}</span>
        </p>

        <div className="flex gap-4">
          <Button onClick={() => setPaperDepositOpen(true)}>
            Deposit (Paper Trading)
          </Button>
          <Button variant="secondary" onClick={() => setStripeDepositOpen(true)}>
            Deposit (Stripe)
          </Button>
        </div>
      </div>

      <DepositDialog open={paperDepositOpen} onOpenChange={setPaperDepositOpen} />
      <StripeDepositDialog open={stripeDepositOpen} onOpenChange={setStripeDepositOpen} />
    </div>
  );
}
