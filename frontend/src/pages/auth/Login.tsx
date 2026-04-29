import { useState, useEffect, useRef } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Turnstile } from '@marsidev/react-turnstile';
import { toast } from 'sonner';
import { ShieldAlert, ArrowLeft } from 'lucide-react';

import { loginSchema, type LoginFormValues } from '@/lib/schemas';
import { authApi } from '@/api/auth';
import { userApi } from '@/api/user';
import { useAuthStore } from '@/stores/authStore';

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
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

type LoginStep = 'credentials' | '2fa';

export function Login() {
  const navigate = useNavigate();
  const { securityAlert, setSecurityAlert, setAccessToken, setUser, setStatus } = useAuthStore();

  const [step, setStep] = useState<LoginStep>('credentials');
  const [turnstileToken, setTurnstileToken] = useState<string | null>(null);
  const [rateLimited, setRateLimited] = useState(false);

  // store credentials for the 2FA re-send
  const credentialsRef = useRef<{ email: string; password: string }>({ email: '', password: '' });

  // 2FA token state
  const [totpCode, setTotpCode] = useState('');
  const [totpError, setTotpError] = useState<string | null>(null);
  const [twoFASubmitting, setTwoFASubmitting] = useState(false);

  // reset security alert after display
  useEffect(() => {
    if (securityAlert) {
      const timer = setTimeout(() => setSecurityAlert(false), 10_000);
      return () => clearTimeout(timer);
    }
  }, [securityAlert, setSecurityAlert]);

  // credentials form
  const form = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: '', password: '' },
  });

  const onCredentialsSubmit = async (data: LoginFormValues) => {
    if (!turnstileToken) {
      toast.error('Anti-bot check failed. Please wait for the Turnstile challenge.');
      return;
    }

    try {
      const response = await authApi.login(data.email, data.password, turnstileToken);

      if (response.data.status === '2fa_required') {
        credentialsRef.current = { email: data.email, password: data.password };
        setStep('2fa');
        return;
      }

      // status === "success"
      const accessToken = response.data.access_token;
      setAccessToken(accessToken);

      const userResponse = await userApi.getUser();
      setUser(userResponse.data);
      setStatus('authenticated');

      navigate('/dashboard');
    } catch (error: any) {
      const status = error.response?.status;

      if (status === 401) {
        form.setError('root', {
          type: 'manual',
          message: 'Invalid email or password.',
        });
      } else if (status === 429) {
        setRateLimited(true);
        setTimeout(() => setRateLimited(false), 5000);
      }
      // other errors (5xx, 423) handled by axios interceptor
    }
  };

  const onTwoFASubmit = async () => {
    setTotpError(null);

    if (!/^\d{6}$/.test(totpCode)) {
      setTotpError('Code must be exactly 6 digits.');
      return;
    }

    setTwoFASubmitting(true);
    try {
      const response = await authApi.verify2FA(
        credentialsRef.current.email,
        credentialsRef.current.password,
        totpCode,
      );

      const accessToken = response.data.access_token;
      setAccessToken(accessToken);

      const userResponse = await userApi.getUser();
      setUser(userResponse.data);
      setStatus('authenticated');

      navigate('/dashboard');
    } catch (error: any) {
      if (error.response?.status === 401) {
        setTotpError('Invalid code. Please try again.');
      } else {
        toast.error('Verification failed. Please try again.');
      }
    } finally {
      setTwoFASubmitting(false);
    }
  };

  const goBackToCredentials = () => {
    setStep('credentials');
    setTotpCode('');
    setTotpError(null);
  };

  // ─── RENDER ───────────────────────────────────────────────────────
  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/30 p-6">
      <div className="w-full max-w-md space-y-6">

        {/* Security alert banner */}
        {securityAlert && (
          <Alert variant="destructive" className="animate-in fade-in slide-in-from-top-4 duration-500">
            <ShieldAlert className="h-4 w-4" />
            <AlertTitle>Security Notice</AlertTitle>
            <AlertDescription>
              Your session was invalidated due to suspicious activity. Please log in again.
            </AlertDescription>
          </Alert>
        )}

        <Card className="shadow-xl border-border/50">
          <CardHeader className="text-center pb-2">
            <CardTitle className="text-3xl font-bold tracking-tight">
              {step === 'credentials' ? 'Welcome back' : 'Two-Factor Authentication'}
            </CardTitle>
            <CardDescription>
              {step === 'credentials'
                ? 'Sign in to your RoboAdvisor account'
                : `Enter the 6-digit code from your authenticator app for ${credentialsRef.current.email}`}
            </CardDescription>
          </CardHeader>

          <CardContent className="space-y-6 pt-4">

            {/* ── STEP 1: Credentials ─────────────────────────── */}
            {step === 'credentials' && (
              <Form {...form}>
                <form onSubmit={form.handleSubmit(onCredentialsSubmit)} className="space-y-5">

                  {/* Root-level form error (invalid credentials) */}
                  {form.formState.errors.root && (
                    <Alert variant="destructive">
                      <AlertDescription>{form.formState.errors.root.message}</AlertDescription>
                    </Alert>
                  )}

                  <FormField
                    control={form.control}
                    name="email"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Email</FormLabel>
                        <FormControl>
                          <Input placeholder="you@example.com" {...field} className="h-11" />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="password"
                    render={({ field }) => (
                      <FormItem>
                        <div className="flex items-center justify-between">
                          <FormLabel>Password</FormLabel>
                          <Link
                            to="/forgot-password"
                            className="text-xs font-medium text-primary hover:text-primary/80 transition-colors"
                          >
                            Forgot password?
                          </Link>
                        </div>
                        <FormControl>
                          <Input type="password" placeholder="••••••••" {...field} className="h-11" />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <div className="flex justify-center overflow-hidden rounded-md border border-border/50 bg-muted/20">
                    <Turnstile
                      siteKey={import.meta.env.VITE_TURNSTILE_SITE_KEY || '1x00000000000000000000AA'}
                      onSuccess={(token) => setTurnstileToken(token)}
                      onError={() => toast.error("Anti-bot check failed. Please try again.")}
                      options={{ theme: 'auto' }}
                    />
                  </div>

                  <Button
                    type="submit"
                    className="w-full h-11 text-base font-semibold"
                    disabled={form.formState.isSubmitting || !turnstileToken || rateLimited}
                  >
                    {rateLimited
                      ? 'Please wait...'
                      : form.formState.isSubmitting
                        ? 'Signing in...'
                        : 'Sign In'}
                  </Button>
                </form>
              </Form>
            )}

            {/* ── STEP 2: 2FA Gate ────────────────────────────── */}
            {step === '2fa' && (
              <div className="space-y-5">
                <div className="space-y-2">
                  <label htmlFor="totp-code" className="text-sm font-medium leading-none">
                    Authentication Code
                  </label>
                  <Input
                    id="totp-code"
                    placeholder="000000"
                    maxLength={6}
                    value={totpCode}
                    onChange={(e) => {
                      // allow only digits
                      const val = e.target.value.replace(/\D/g, '');
                      setTotpCode(val);
                      setTotpError(null);
                    }}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault();
                        onTwoFASubmit();
                      }
                    }}
                    className={`h-12 text-center text-2xl tracking-[0.5em] font-mono ${
                      totpError ? 'border-destructive focus-visible:ring-destructive' : ''
                    }`}
                    autoFocus
                  />
                  {totpError && (
                    <p className="text-sm font-medium text-destructive">{totpError}</p>
                  )}
                </div>

                <Button
                  onClick={onTwoFASubmit}
                  className="w-full h-11 text-base font-semibold"
                  disabled={twoFASubmitting || totpCode.length !== 6}
                >
                  {twoFASubmitting ? 'Verifying...' : 'Verify Code'}
                </Button>

                <Button
                  variant="ghost"
                  className="w-full text-muted-foreground"
                  onClick={goBackToCredentials}
                >
                  <ArrowLeft className="mr-2 h-4 w-4" />
                  Back to login
                </Button>
              </div>
            )}
          </CardContent>
        </Card>

        {step === 'credentials' && (
          <p className="text-center text-sm text-muted-foreground">
            Don&apos;t have an account?{' '}
            <Link to="/register" className="font-semibold text-primary hover:text-primary/80 transition-colors">
              Create one
            </Link>
          </p>
        )}
      </div>
    </div>
  );
}
