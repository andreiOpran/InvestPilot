import { Link } from 'react-router-dom';
import { TrendingUp, Clock, ClipboardList } from 'lucide-react';

import { TwoFactorSetup } from '@/components/security/TwoFactorSetup';
import { useAuthStore } from '@/stores/authStore';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';

const riskLabels: Record<number, string> = {
  1: 'Very Conservative',
  2: 'Conservative',
  3: 'Balanced',
  4: 'Growth',
  5: 'Aggressive Growth',
};

export function Settings() {
  const { user } = useAuthStore();

  return (
    <div className="max-w-2xl mx-auto py-8 px-4 space-y-8">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Settings</h1>
        <p className="text-muted-foreground mt-1">Manage your account security and investment profile.</p>
      </div>

      {/* Investment Profile Card */}
      <Card>
        <CardHeader>
          <CardTitle>Investment Profile</CardTitle>
          <CardDescription>Your personalized risk and horizon settings computed from the onboarding questionnaire.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          {user && user.risk_tolerance > 0 ? (
            <>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                {/* Risk Tolerance */}
                <div className="rounded-xl border border-border/60 bg-muted/30 p-4 space-y-1">
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <TrendingUp className="h-4 w-4" />
                    Risk Tolerance
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-lg font-bold">{riskLabels[user.risk_tolerance] ?? 'Unknown'}</span>
                    <Badge variant="outline" className="text-xs">Level {user.risk_tolerance}/5</Badge>
                  </div>
                </div>

                {/* Investment Horizon */}
                <div className="rounded-xl border border-border/60 bg-muted/30 p-4 space-y-1">
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Clock className="h-4 w-4" />
                    Investment Horizon
                  </div>
                  <p className="text-lg font-bold">{user.investment_horizon} years</p>
                </div>
              </div>

              <Separator />

              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium">Retake Questionnaire</p>
                  <p className="text-xs text-muted-foreground mt-0.5">Update your profile by re-answering the onboarding questions.</p>
                </div>
                <Button variant="outline" asChild className="gap-2 shrink-0">
                  <Link to="/onboarding?edit=true">
                    <ClipboardList className="h-4 w-4" />
                    Retake
                  </Link>
                </Button>
              </div>
            </>
          ) : (
            <div className="flex flex-col items-center gap-4 py-4 text-center">
              <p className="text-muted-foreground text-sm">
                You haven&apos;t completed your investment profile yet.
              </p>
              <Button asChild className="gap-2">
                <Link to="/onboarding">
                  <ClipboardList className="h-4 w-4" />
                  Start Questionnaire
                </Link>
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      <TwoFactorSetup />
    </div>
  );
}

