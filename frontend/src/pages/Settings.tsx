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
    <div className="p-4 md:p-8 space-y-6 max-w-3xl mx-auto">

      {/* Header */}
      <div className="space-y-0.5">
        <h1 className="text-xl font-semibold tracking-tight">Settings</h1>
        <p className="text-sm text-muted-foreground">
          Manage your account security and investment profile.
        </p>
      </div>

      {/* Investment Profile */}
      <Card>
        <CardHeader className="pb-4">
          <div className="flex items-start justify-between">
            <div>
              <CardTitle className="text-sm font-semibold tracking-tight">Investment Profile</CardTitle>
              <CardDescription className="text-xs mt-0.5">
                Risk and horizon settings computed from your onboarding questionnaire.
              </CardDescription>
            </div>
            {/* {user && user.risk_tolerance > 0 && (
              <Button variant="outline" asChild className="gap-1.5 h-8 text-xs shrink-0">
                <Link to="/onboarding?edit=true">
                  <ClipboardList className="h-3.5 w-3.5" />
                  Retake
                </Link>
              </Button>
            )} */}
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {user && user.risk_tolerance > 0 ? (
            <>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div className="rounded-lg border bg-muted/30 p-4 space-y-1.5">
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <TrendingUp className="h-3.5 w-3.5" />
                    Risk Tolerance
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="font-semibold text-sm">{riskLabels[user.risk_tolerance] ?? 'Unknown'}</span>
                    <Badge variant="outline" className="text-[10px] h-4 px-1.5">
                      Level {user.risk_tolerance}/5
                    </Badge>
                  </div>
                </div>

                <div className="rounded-lg border bg-muted/30 p-4 space-y-1.5">
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <Clock className="h-3.5 w-3.5" />
                    Investment Horizon
                  </div>
                  <p className="font-semibold text-sm">{user.investment_horizon} years</p>
                </div>
              </div>

              <Separator />

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <p className="text-sm font-medium">Retake questionnaire</p>
                  <p className="text-xs text-muted-foreground">Update your profile by re-answering the onboarding questions.</p>
                </div>
                <Button variant="outline" asChild className="gap-1.5 h-8 text-xs shrink-0">
                  <Link to="/onboarding?edit=true">
                    <ClipboardList className="h-3.5 w-3.5" />
                    Retake
                  </Link>
                </Button>
              </div>
            </>
          ) : (
            <div className="flex flex-col items-center gap-3 py-6 text-center">
              <p className="text-sm text-muted-foreground">
                You haven&apos;t completed your investment profile yet.
              </p>
              <Button asChild className="gap-1.5 h-9 text-sm font-medium">
                <Link to="/onboarding">
                  <ClipboardList className="h-3.5 w-3.5" />
                  Start questionnaire
                </Link>
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Security */}
      <TwoFactorSetup />
    </div>
  );
}
