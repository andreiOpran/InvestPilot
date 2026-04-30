import { Link } from 'react-router-dom';
import { MailCheck, Landmark } from 'lucide-react';

import { Button } from '@/components/ui/button';

export function RegisterSuccess() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-muted/30 px-4 py-12">
      <div className="w-full max-w-sm space-y-6">

        {/* Logo */}
        <div className="flex flex-col items-center gap-2 text-center">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl border bg-background shadow-sm">
            <Landmark className="h-5 w-5 text-primary" />
          </div>
          <p className="text-sm font-semibold tracking-tight">RoboAdvisor</p>
        </div>

        {/* Card */}
        <div className="rounded-xl border bg-card shadow-sm p-8">
          <div className="flex flex-col items-center text-center space-y-4">
            <div className="flex h-14 w-14 items-center justify-center rounded-full bg-primary/10">
              <MailCheck className="h-7 w-7 text-primary" />
            </div>
            <div className="space-y-1.5">
              <p className="font-semibold tracking-tight">Check your inbox</p>
              <p className="text-xs text-muted-foreground leading-relaxed max-w-xs">
                If the email you provided is valid, we&apos;ve sent a verification link.
                Please check your inbox and click the link to activate your account.
              </p>
            </div>
            <Button asChild variant="outline" className="w-full h-10 font-medium mt-2">
              <Link to="/login">Go to login</Link>
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
