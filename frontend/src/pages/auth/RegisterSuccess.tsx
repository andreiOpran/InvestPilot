import { Link } from 'react-router-dom';
import { MailCheck } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

export function RegisterSuccess() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/30 p-6">
      <Card className="w-full max-w-md shadow-xl border-border/50">
        <CardHeader className="text-center pb-2">
          <div className="flex justify-center mb-4">
            <MailCheck className="h-20 w-20 text-primary" />
          </div>
          <CardTitle className="text-2xl font-bold tracking-tight">Check your inbox</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-center space-y-6 pt-4 pb-8">
          <p className="text-center text-muted-foreground text-sm">
            If the email you provided is valid, we&apos;ve sent a verification link.
            Please check your inbox and click the link to activate your account.
          </p>
          <Button asChild variant="outline" className="w-full h-11 text-base">
            <Link to="/login">Go to Login</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
