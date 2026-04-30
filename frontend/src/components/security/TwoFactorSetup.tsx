import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { toast } from 'sonner';
import { ShieldCheck, Copy, Check } from 'lucide-react';

import { enable2FASchema, type Enable2FAFormValues } from '@/lib/schemas';
import { authApi } from '@/api/auth';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Separator } from '@/components/ui/separator';

interface Setup2FAData {
  secret: string;
  uri: string;
  qr_code_b64: string;
}

type ViewState = 'loading' | 'setup' | 'already_enabled' | 'error';

export function TwoFactorSetup() {
  const [viewState, setViewState] = useState<ViewState>('loading');
  const [setupData, setSetupData] = useState<Setup2FAData | null>(null);
  const [copied, setCopied] = useState(false);
  const [enabled, setEnabled] = useState(false);

  const form = useForm<Enable2FAFormValues>({
    resolver: zodResolver(enable2FASchema),
    defaultValues: { token: '' },
  });

  useEffect(() => {
    let isMounted = true;

    authApi.setup2FA()
      .then((res) => {
        if (isMounted) {
          setSetupData(res.data);
          setViewState('setup');
        }
      })
      .catch((err) => {
        if (!isMounted) return;
        const msg: string = err.response?.data?.error ?? '';
        if (err.response?.status === 400 && msg.toLowerCase().includes('already enabled')) {
          setViewState('already_enabled');
        } else {
          setViewState('error');
        }
      });

    return () => { isMounted = false; };
  }, []);

  const copySecret = async () => {
    if (!setupData?.secret) return;
    await navigator.clipboard.writeText(setupData.secret);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const onSubmit = async (data: Enable2FAFormValues) => {
    try {
      await authApi.enable2FA(data.token);
      toast.success('2FA enabled successfully');
      setEnabled(true);
    } catch (err: any) {
      const msg: string = err.response?.data?.error ?? '';
      if (err.response?.status === 400 && msg.toLowerCase().includes('already enabled')) {
        setViewState('already_enabled');
      } else if (err.response?.status === 400) {
        form.setError('token', { type: 'manual', message: 'Incorrect code. Authenticator not linked.' });
      } else {
        toast.error('Something went wrong. Please try again.');
      }
    }
  };

  // Loading
  if (viewState === 'loading') {
    return (
      <Card>
        <CardHeader className="pb-4">
          <Skeleton className="h-5 w-48" />
          <Skeleton className="h-3.5 w-72 mt-1" />
        </CardHeader>
        <CardContent className="space-y-4">
          <Skeleton className="h-44 w-44 mx-auto rounded-lg" />
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-9 w-full" />
        </CardContent>
      </Card>
    );
  }

  // Already enabled
  if (viewState === 'already_enabled') {
    return (
      <Card>
        <CardHeader className="pb-4">
          <div className="flex items-center gap-2">
            <CardTitle className="text-sm font-semibold tracking-tight">Two-Factor Authentication</CardTitle>
            <Badge variant="secondary" className="text-emerald-600 bg-emerald-500/10 text-[10px] h-4 px-1.5">
              Active
            </Badge>
          </div>
          <CardDescription className="text-xs">
            Your account is protected with two-factor authentication.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Alert>
            <ShieldCheck className="h-4 w-4 text-emerald-500" />
            <AlertTitle className="text-sm font-medium">2FA is active</AlertTitle>
            <AlertDescription className="text-xs">
              Every login requires a 6-digit code from your authenticator app.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    );
  }

  // Error
  if (viewState === 'error') {
    return (
      <Card>
        <CardHeader className="pb-4">
          <CardTitle className="text-sm font-semibold tracking-tight">Two-Factor Authentication</CardTitle>
        </CardHeader>
        <CardContent>
          <Alert variant="destructive">
            <AlertDescription className="text-xs">
              Failed to load 2FA setup. Please refresh the page and try again.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    );
  }

  // Successfully enabled
  if (enabled) {
    return (
      <Card>
        <CardHeader className="pb-4">
          <div className="flex items-center gap-2">
            <CardTitle className="text-sm font-semibold tracking-tight">Two-Factor Authentication</CardTitle>
            <Badge variant="secondary" className="text-emerald-600 bg-emerald-500/10 text-[10px] h-4 px-1.5">
              Active
            </Badge>
          </div>
          <CardDescription className="text-xs">Your account is now protected with 2FA.</CardDescription>
        </CardHeader>
        <CardContent>
          <Alert>
            <ShieldCheck className="h-4 w-4 text-emerald-500" />
            <AlertTitle className="text-sm font-medium">2FA enabled successfully</AlertTitle>
            <AlertDescription className="text-xs">
              Every future login will require a 6-digit code from your authenticator app.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    );
  }

  // Setup form
  return (
    <Card>
      <CardHeader className="pb-4">
        <CardTitle className="text-sm font-semibold tracking-tight">Two-Factor Authentication</CardTitle>
        <CardDescription className="text-xs">
          Add an extra layer of security using a TOTP authenticator app (e.g. Google Authenticator, Authy).
        </CardDescription>
      </CardHeader>
      <Separator />

      <CardContent className="pt-6 space-y-0">

        {/* Step 1 — Scan QR */}
        <div className="space-y-4 pb-6">
          <div className="flex items-center gap-3">
            <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-primary text-[10px] font-bold text-primary-foreground">
              1
            </span>
            <div>
              <p className="text-sm font-medium">Scan the QR code</p>
              <p className="text-xs text-muted-foreground mt-0.5">
                Open your authenticator app and scan the code below.
              </p>
            </div>
          </div>

          {setupData?.qr_code_b64 && (
            <div className="flex justify-center py-2">
              <div className="rounded-xl border bg-white p-3 shadow-sm">
                <img
                  src={setupData.qr_code_b64}
                  alt="Scan with authenticator app"
                  className="h-44 w-44 rounded-md block"
                />
              </div>
            </div>
          )}
        </div>

        <Separator />

        {/* Step 2 — Manual entry */}
        <div className="space-y-4 py-6">
          <div className="flex items-center gap-3">
            <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-primary text-[10px] font-bold text-primary-foreground">
              2
            </span>
            <div>
              <p className="text-sm font-medium">Or enter the secret manually</p>
              <p className="text-xs text-muted-foreground mt-0.5">
                If you can&apos;t scan the QR code, type this key into your authenticator app.
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <code className="flex-1 rounded-lg border bg-muted/30 px-3 py-2.5 font-mono text-xs tracking-wider select-all break-all leading-relaxed">
              {setupData?.secret}
            </code>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={copySecret}
              className="shrink-0 h-9 w-9 p-0"
            >
              {copied ? <Check className="h-3.5 w-3.5 text-emerald-500" /> : <Copy className="h-3.5 w-3.5" />}
            </Button>
          </div>
        </div>

        <Separator />

        {/* Step 3 — Confirm code */}
        <div className="space-y-4 pt-6">
          <div className="flex items-center gap-3">
            <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-primary text-[10px] font-bold text-primary-foreground">
              3
            </span>
            <div>
              <p className="text-sm font-medium">Enter the confirmation code</p>
              <p className="text-xs text-muted-foreground mt-0.5">
                Enter the 6-digit code from your authenticator app to confirm the link.
              </p>
            </div>
          </div>

          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col items-center gap-4">
              <FormField
                control={form.control}
                name="token"
                render={({ field }) => (
                  <FormItem className="flex flex-col items-center text-center">
                    <FormLabel className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                      Confirmation Code
                    </FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        placeholder="000000"
                        maxLength={6}
                        onChange={(e) => field.onChange(e.target.value.replace(/\D/g, ''))}
                        className="h-11 w-40 text-center text-xl tracking-[0.4em] font-mono"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <Button
                type="submit"
                disabled={form.formState.isSubmitting}
                className="h-9 px-6 text-sm font-medium"
              >
                {form.formState.isSubmitting ? 'Verifying...' : 'Enable 2FA'}
              </Button>
            </form>
          </Form>
        </div>

      </CardContent>
    </Card>
  );
}
