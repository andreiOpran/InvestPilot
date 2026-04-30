import { useState, useEffect, useRef } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Turnstile } from '@marsidev/react-turnstile';
import { toast } from 'sonner';
import { ShieldAlert, ArrowLeft, Landmark } from 'lucide-react';

import { loginSchema, type LoginFormValues } from '@/lib/schemas';
import { authApi } from '@/api/auth';
import { userApi } from '@/api/user';
import { useAuthStore } from '@/stores/authStore';

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
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

  const credentialsRef = useRef<{ email: string; password: string }>({ email: '', password: '' });

  const [totpCode, setTotpCode] = useState('');
  const [totpError, setTotpError] = useState<string | null>(null);
  const [twoFASubmitting, setTwoFASubmitting] = useState(false);

  useEffect(() => {
    if (securityAlert) {
      const timer = setTimeout(() => setSecurityAlert(false), 10_000);
      return () => clearTimeout(timer);
    }
  }, [securityAlert, setSecurityAlert]);

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

      const accessToken = response.data.access_token;
      setAccessToken(accessToken);

      const userResponse = await userApi.getUser();
      setUser(userResponse.data);
      setStatus('authenticated');

      navigate('/dashboard');
    } catch (error: any) {
      const status = error.response?.status;
      const serverMsg: string = error.response?.data?.error || error.response?.data?.message || '';

      if (status === 401) {
        form.setError('root', { type: 'manual', message: 'Invalid email or password.' });
      } else if (status === 403) {
        form.setError('root', { type: 'manual', message: 'Anti-bot verification failed. Please wait for the challenge to reload.' });
      } else if (status === 423) {
        form.setError('root', { type: 'manual', message: serverMsg || 'Your account is temporarily locked. Please try again in 15 minutes.' });
      } else if (status === 429) {
        form.setError('root', { type: 'manual', message: serverMsg || 'Too many failed attempts. Please wait before trying again.' });
        setRateLimited(true);
        setTimeout(() => setRateLimited(false), 5000);
      } else if (!error.response) {
        form.setError('root', { type: 'manual', message: 'Unable to reach the server. Check your connection and try again.' });
      }
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
      const status = error.response?.status;
      if (status === 401) {
        setTotpError('Incorrect code. Please check your authenticator app and try again.');
      } else if (status === 429 || status === 423) {
        setTotpError('Too many failed attempts. Your account has been temporarily locked.');
      } else if (!error.response) {
        setTotpError('Unable to reach the server. Check your connection and try again.');
      } else {
        setTotpError('Verification failed. Please go back and try again.');
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

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-muted/30 px-4 py-12">
      <div className="w-full max-w-sm space-y-6">

        {/* Logo */}
        <div className="flex flex-col items-center gap-2 text-center">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl border bg-background shadow-sm">
            <Landmark className="h-5 w-5 text-primary" />
          </div>
          <div>
            <p className="text-sm font-semibold tracking-tight">RoboAdvisor</p>
            <p className="text-xs text-muted-foreground mt-0.5">
              {step === 'credentials' ? 'Sign in to your account' : 'Two-factor authentication'}
            </p>
          </div>
        </div>

        {/* Security alert */}
        {securityAlert && (
          <Alert variant="destructive" className="animate-in fade-in slide-in-from-top-4 duration-500">
            <ShieldAlert className="h-4 w-4" />
            <AlertTitle>Security Notice</AlertTitle>
            <AlertDescription>
              Your session was invalidated due to suspicious activity. Please log in again.
            </AlertDescription>
          </Alert>
        )}

        {/* Card */}
        <div className="rounded-xl border bg-card shadow-sm p-6 space-y-5">

          {/* Credentials step */}
          {step === 'credentials' && (
            <Form {...form}>
              <form onSubmit={form.handleSubmit(onCredentialsSubmit)} className="space-y-4">

                {form.formState.errors.root ? (
                  <Alert variant="destructive">
                    <AlertDescription>{form.formState.errors.root.message}</AlertDescription>
                  </Alert>
                ) : (form.formState.errors.email || form.formState.errors.password) ? (
                  <Alert variant="destructive">
                    <AlertDescription className="space-y-1">
                      {form.formState.errors.email?.message && <p>{form.formState.errors.email.message}</p>}
                      {form.formState.errors.password?.message && <p>{form.formState.errors.password.message}</p>}
                    </AlertDescription>
                  </Alert>
                ) : null}

                <FormField
                  control={form.control}
                  name="email"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Email</FormLabel>
                      <FormControl>
                        <Input placeholder="you@example.com" {...field} className="h-10" />
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
                        <FormLabel className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Password</FormLabel>
                        <Link
                          to="/forgot-password"
                          className="text-xs text-primary hover:text-primary/80 transition-colors"
                        >
                          Forgot password?
                        </Link>
                      </div>
                      <FormControl>
                        <Input type="password" placeholder="••••••••" {...field} className="h-10" />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <div className="flex justify-center overflow-hidden rounded-lg border border-border/50 bg-muted/20">
                  <Turnstile
                    siteKey={import.meta.env.VITE_TURNSTILE_SITE_KEY || '1x00000000000000000000AA'}
                    onSuccess={(token) => setTurnstileToken(token)}
                    onError={() => toast.error('Anti-bot check failed. Please try again.')}
                    options={{ theme: 'auto' }}
                  />
                </div>

                <Button
                  type="submit"
                  className="w-full h-10 font-medium"
                  disabled={form.formState.isSubmitting || !turnstileToken || rateLimited}
                >
                  {rateLimited ? 'Please wait...' : form.formState.isSubmitting ? 'Signing in...' : 'Sign in'}
                </Button>
              </form>
            </Form>
          )}

          {/* 2FA step */}
          {step === '2fa' && (
            <div className="space-y-4">
              <div className="space-y-3">
                <div className="space-y-1">
                  <label htmlFor="totp-code" className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                    Authentication Code
                  </label>
                  <p className="text-xs text-muted-foreground leading-relaxed">
                    Enter the 6-digit code from your authenticator app for{' '}
                    <span className="font-medium text-foreground break-all">{credentialsRef.current.email}</span>
                  </p>
                </div>
                <Input
                  id="totp-code"
                  placeholder="000000"
                  maxLength={6}
                  value={totpCode}
                  onChange={(e) => {
                    const val = e.target.value.replace(/\D/g, '');
                    setTotpCode(val);
                    setTotpError(null);
                  }}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') { e.preventDefault(); onTwoFASubmit(); }
                  }}
                  className={`h-12 text-center text-2xl tracking-[0.5em] font-mono ${
                    totpError ? 'border-destructive focus-visible:ring-destructive' : ''
                  }`}
                  autoFocus
                />
                {totpError && <p className="text-xs font-medium text-destructive">{totpError}</p>}
              </div>

              <Button
                onClick={onTwoFASubmit}
                className="w-full h-10 font-medium"
                disabled={twoFASubmitting || totpCode.length !== 6}
              >
                {twoFASubmitting ? 'Verifying...' : 'Verify code'}
              </Button>

              <Button
                variant="ghost"
                className="w-full text-muted-foreground text-sm"
                onClick={goBackToCredentials}
              >
                <ArrowLeft className="mr-2 h-3.5 w-3.5" />
                Back to login
              </Button>
            </div>
          )}
        </div>

        {step === 'credentials' && (
          <p className="text-center text-xs text-muted-foreground">
            Don&apos;t have an account?{' '}
            <Link to="/register" className="font-medium text-foreground hover:text-primary transition-colors">
              Create one
            </Link>
          </p>
        )}
      </div>
    </div>
  );
}
