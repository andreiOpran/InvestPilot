import { useEffect, useState } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { CheckCircle2, XCircle, Loader2 } from 'lucide-react';

import { authApi } from '@/api/auth';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

export function VerifyEmail() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token');
  
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading');

  useEffect(() => {
    let isMounted = true;

    if (!token) {
      setStatus('error');
      return;
    }

    const verify = async () => {
      try {
        await authApi.verifyEmail(token);
        if (isMounted) setStatus('success');
      } catch (error) {
        if (isMounted) setStatus('error');
      }
    };

    verify();

    return () => {
      isMounted = false;
    };
  }, [token]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/30 p-6">
      <Card className="w-full max-w-md shadow-xl border-border/50">
        <CardHeader className="text-center pb-2">
          <CardTitle className="text-2xl font-bold tracking-tight">Email Verification</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-center justify-center space-y-6 pt-6 pb-8">
          
          {status === 'loading' && (
            <div className="flex flex-col items-center space-y-4 animate-in fade-in zoom-in duration-300">
              <Loader2 className="h-16 w-16 text-primary animate-spin" />
              <CardDescription className="text-base">Verifying your email address...</CardDescription>
            </div>
          )}

          {status === 'success' && (
            <div className="flex flex-col items-center space-y-6 animate-in fade-in slide-in-from-bottom-4 duration-500">
              <CheckCircle2 className="h-20 w-20 text-emerald-500" />
              <div className="space-y-2 text-center">
                <h3 className="text-xl font-semibold text-foreground">Verification Complete</h3>
                <p className="text-muted-foreground">Your email has been successfully verified.</p>
              </div>
              <Button asChild className="w-full h-11 text-base font-semibold shadow-md">
                <Link to="/login">Go to Login</Link>
              </Button>
            </div>
          )}

          {status === 'error' && (
            <div className="flex flex-col items-center space-y-6 animate-in fade-in slide-in-from-bottom-4 duration-500">
              <XCircle className="h-20 w-20 text-destructive" />
              <div className="space-y-2 text-center">
                <h3 className="text-xl font-semibold text-foreground">Verification Failed</h3>
                <p className="text-muted-foreground text-sm">
                  The link is invalid, has expired, or is missing a token.
                </p>
              </div>
              <Button asChild variant="outline" className="w-full h-11 text-base shadow-sm">
                <Link to="/register">Return to Register</Link>
              </Button>
            </div>
          )}

        </CardContent>
      </Card>
    </div>
  );
}
