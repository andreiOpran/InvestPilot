import { useEffect, useState } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { CheckCircle2, XCircle, Loader2, Landmark } from 'lucide-react';

import { authApi } from '@/api/auth';
import { Button } from '@/components/ui/button';

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

    return () => { isMounted = false; };
  }, [token]);

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

            {status === 'loading' && (
              <>
                <div className="flex h-14 w-14 items-center justify-center rounded-full bg-primary/10">
                  <Loader2 className="h-7 w-7 text-primary animate-spin" />
                </div>
                <div className="space-y-1">
                  <p className="font-semibold tracking-tight">Verifying email</p>
                  <p className="text-xs text-muted-foreground">Please wait a moment...</p>
                </div>
              </>
            )}

            {status === 'success' && (
              <div className="flex flex-col items-center space-y-4 animate-in fade-in slide-in-from-bottom-4 duration-500">
                <div className="flex h-14 w-14 items-center justify-center rounded-full bg-emerald-500/10">
                  <CheckCircle2 className="h-7 w-7 text-emerald-500" />
                </div>
                <div className="space-y-1.5">
                  <p className="font-semibold tracking-tight">Email verified</p>
                  <p className="text-xs text-muted-foreground leading-relaxed">
                    Your email has been successfully verified. You can now sign in.
                  </p>
                </div>
                <Button asChild className="w-full h-10 font-medium mt-2">
                  <Link to="/login">Go to login</Link>
                </Button>
              </div>
            )}

            {status === 'error' && (
              <div className="flex flex-col items-center space-y-4 animate-in fade-in slide-in-from-bottom-4 duration-500">
                <div className="flex h-14 w-14 items-center justify-center rounded-full bg-destructive/10">
                  <XCircle className="h-7 w-7 text-destructive" />
                </div>
                <div className="space-y-1.5">
                  <p className="font-semibold tracking-tight">Verification failed</p>
                  <p className="text-xs text-muted-foreground leading-relaxed">
                    The link is invalid, has expired, or is missing a token.
                  </p>
                </div>
                <Button asChild variant="outline" className="w-full h-10 font-medium mt-2">
                  <Link to="/register">Back to register</Link>
                </Button>
              </div>
            )}

          </div>
        </div>
      </div>
    </div>
  );
}
