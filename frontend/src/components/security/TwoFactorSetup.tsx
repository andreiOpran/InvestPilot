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
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';

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
        form.setError('token', {
          type: 'manual',
          message: 'Incorrect code. Authenticator not linked.',
        });
      } else {
        toast.error('Something went wrong. Please try again.');
      }
    }
  };

  // ─── Loading skeleton ────────────────────────────────────────────
  if (viewState === 'loading') {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-4 w-72 mt-1" />
        </CardHeader>
        <CardContent className="space-y-6">
          <Skeleton className="h-48 w-48 mx-auto rounded-lg" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </CardContent>
      </Card>
    );
  }

  // ─── Already enabled ─────────────────────────────────────────────
  if (viewState === 'already_enabled') {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            Two-Factor Authentication
            <Badge variant="secondary" className="text-emerald-600 bg-emerald-500/10">
              Active
            </Badge>
          </CardTitle>
          <CardDescription>
            Your account is protected with two-factor authentication.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Alert>
            <ShieldCheck className="h-4 w-4 text-emerald-500" />
            <AlertTitle>2FA is already active on your account</AlertTitle>
            <AlertDescription>
              You are already enrolled in two-factor authentication. Every login requires
              a 6-digit code from your authenticator app.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    );
  }

  // ─── Error state ─────────────────────────────────────────────────
  if (viewState === 'error') {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Two-Factor Authentication</CardTitle>
        </CardHeader>
        <CardContent>
          <Alert variant="destructive">
            <AlertDescription>
              Failed to load 2FA setup. Please refresh the page and try again.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    );
  }

  // ─── Successfully enabled ────────────────────────────────────────
  if (enabled) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            Two-Factor Authentication
            <Badge variant="secondary" className="text-emerald-600 bg-emerald-500/10">
              Active
            </Badge>
          </CardTitle>
          <CardDescription>Your account is now protected with 2FA.</CardDescription>
        </CardHeader>
        <CardContent>
          <Alert>
            <ShieldCheck className="h-4 w-4 text-emerald-500" />
            <AlertTitle>2FA enabled successfully</AlertTitle>
            <AlertDescription>
              Every future login will require a 6-digit code from your authenticator app.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    );
  }

  // ─── Setup form ──────────────────────────────────────────────────
  return (
    <Card>
      <CardHeader>
        <CardTitle>Two-Factor Authentication</CardTitle>
        <CardDescription>
          Add an extra layer of security using a TOTP authenticator app (e.g. Google Authenticator, Authy).
        </CardDescription>
      </CardHeader>

      <CardContent className="space-y-6">

        {/* Step 1 — Scan QR */}
        <div className="rounded-lg border border-border/60 bg-muted/20 p-5 space-y-4">
          <div className="flex items-center gap-3">
            <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-[11px] font-bold text-primary-foreground">1</span>
            <h3 className="text-sm font-semibold">Scan the QR code</h3>
          </div>
          <p className="text-sm text-muted-foreground">
            Open your authenticator app and scan the QR code below to add your account.
          </p>
          {setupData?.qr_code_b64 && (
            <div className="flex justify-center pt-1">
              <div className="rounded-xl border border-border/50 bg-white p-3 shadow-sm">
                <img
                  src={setupData.qr_code_b64}
                  alt="Scan with authenticator app"
                  className="h-44 w-44 rounded-md"
                />
              </div>
            </div>
          )}
        </div>

        {/* Step 2 — Manual entry */}
        <div className="rounded-lg border border-border/60 bg-muted/20 p-5 space-y-4">
          <div className="flex items-center gap-3">
            <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-[11px] font-bold text-primary-foreground">2</span>
            <h3 className="text-sm font-semibold">Or enter the secret manually</h3>
          </div>
          <p className="text-sm text-muted-foreground">
            If you can&apos;t scan the QR code, enter this secret key into your authenticator app.
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded-md border border-border/60 bg-background px-3 py-2 font-mono text-sm tracking-wider select-all break-all">
              {setupData?.secret}
            </code>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={copySecret}
              className="shrink-0 h-9 w-9 p-0"
            >
              {copied ? <Check className="h-4 w-4 text-emerald-500" /> : <Copy className="h-4 w-4" />}
            </Button>
          </div>
        </div>

        {/* Step 3 — Confirm code */}
        <div className="rounded-lg border border-border/60 bg-muted/20 p-5 space-y-4">
          <div className="flex items-center gap-3">
            <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-[11px] font-bold text-primary-foreground">3</span>
            <h3 className="text-sm font-semibold">Enter the confirmation code</h3>
          </div>
          <p className="text-sm text-muted-foreground">
            Enter the 6-digit code shown in your authenticator app to confirm the link.
          </p>

          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
              <FormField
                control={form.control}
                name="token"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Confirmation Code</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        placeholder="000000"
                        maxLength={6}
                        onChange={(e) => {
                          field.onChange(e.target.value.replace(/\D/g, ''));
                        }}
                        className="h-11 max-w-[160px] text-center text-xl tracking-[0.4em] font-mono"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <Button
                type="submit"
                disabled={form.formState.isSubmitting}
                className="h-10 px-6 font-semibold"
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
