import { useState } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { ShieldCheck, XCircle, Navigation } from 'lucide-react';

import { resetPasswordSchema, type ResetPasswordFormValues } from '@/lib/schemas';
import { authApi } from '@/api/auth';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { PasswordInput } from '@/components/ui/password-input';
import { PasswordRequirements } from '@/components/ui/password-requirements';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';

export function ResetPassword() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token');

  const [status, setStatus] = useState<'idle' | 'success' | 'invalid_token'>('idle');
  const [passwordFocused, setPasswordFocused] = useState(false);

  const form = useForm<ResetPasswordFormValues>({
    resolver: zodResolver(resetPasswordSchema),
    mode: 'onTouched',
    defaultValues: { token: token || '', newPassword: '', confirmPassword: '' },
  });

  const onSubmit = async (data: ResetPasswordFormValues) => {
    if (!token) { setStatus('invalid_token'); return; }

    try {
      await authApi.resetPassword(data.token, data.newPassword);
      setStatus('success');
    } catch (error: any) {
      if (error.response?.status === 400 && error.response.data?.error?.toLowerCase().includes('token')) {
        setStatus('invalid_token');
      } else if (error.response?.status === 400 && error.response.data?.error) {
        form.setError('newPassword', { type: 'manual', message: error.response.data.error.charAt(0).toUpperCase() + error.response.data.error.slice(1) + '.' });
      } else {
        form.setError('root', { type: 'manual', message: 'An unexpected error occurred. Please try again.' });
      }
    }
  };

  const StatusCard = ({ icon, iconClass, title, description, action }: {
    icon: React.ReactNode;
    iconClass: string;
    title: string;
    description: string;
    action: React.ReactNode;
  }) => (
    <div className="rounded-xl border bg-card shadow-sm p-8">
      <div className="flex flex-col items-center text-center space-y-4">
        <div className={`flex h-14 w-14 items-center justify-center rounded-full ${iconClass}`}>
          {icon}
        </div>
        <div className="space-y-1.5">
          <p className="font-semibold tracking-tight">{title}</p>
          <p className="text-xs text-muted-foreground leading-relaxed max-w-xs">{description}</p>
        </div>
        {action}
      </div>
    </div>
  );

  if (!token || status === 'invalid_token') {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center bg-muted/30 px-4 py-12">
        <div className="w-full max-w-sm space-y-6">
          <div className="flex flex-col items-center gap-2 text-center">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl border bg-background shadow-sm">
              <Navigation className="h-5 w-5 text-primary" />
            </div>
            <p className="text-sm font-semibold tracking-tight">InvestPilot</p>
          </div>
          <StatusCard
            icon={<XCircle className="h-7 w-7 text-destructive" />}
            iconClass="bg-destructive/10"
            title="Link expired"
            description="This password reset link is invalid or has expired. Request a new one to continue."
            action={
              <Button asChild className="w-full h-10 font-medium mt-2">
                <Link to="/forgot-password">Request new link</Link>
              </Button>
            }
          />
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-muted/30 px-4 py-12">
      <div className="w-full max-w-sm space-y-6">

        {/* Logo */}
        <div className="flex flex-col items-center gap-2 text-center">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl border bg-background shadow-sm">
            <Navigation className="h-5 w-5 text-primary" />
          </div>
          <div>
            <p className="text-sm font-semibold tracking-tight">InvestPilot</p>
            <p className="text-xs text-muted-foreground mt-0.5">Set new password</p>
          </div>
        </div>

        {status === 'idle' && (
          <div className="rounded-xl border bg-card shadow-sm p-6 space-y-5">
            <p className="text-xs text-muted-foreground text-center pb-4">
              Please enter a strong password for your account.
            </p>

            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                {form.formState.submitCount > 0 && (form.formState.errors.newPassword || form.formState.errors.confirmPassword || form.formState.errors.root) && (
                  <Alert variant="destructive">
                    <AlertDescription className="space-y-1">
                      {form.formState.errors.newPassword?.message && <p>{form.formState.errors.newPassword.message}{form.formState.errors.newPassword.message.endsWith('.') ? '' : '.'}</p>}
                      {form.formState.errors.confirmPassword?.message && <p>{form.formState.errors.confirmPassword.message}{form.formState.errors.confirmPassword.message.endsWith('.') ? '' : '.'}</p>}
                      {form.formState.errors.root?.message && <p>{form.formState.errors.root.message}{form.formState.errors.root.message.endsWith('.') ? '' : '.'}</p>}
                    </AlertDescription>
                  </Alert>
                )}

                <FormField
                  control={form.control}
                  name="newPassword"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className="text-xs font-medium text-muted-foreground uppercase tracking-wide">New Password</FormLabel>
                      <FormControl>
                        <PasswordInput
                          placeholder="••••••••"
                          {...field}
                          className="h-10"
                          onFocus={() => setPasswordFocused(true)}
                          onBlur={() => { setPasswordFocused(false); field.onBlur(); }}
                        />
                      </FormControl>
                      {(passwordFocused || field.value.length > 0) && (
                        <PasswordRequirements password={field.value} />
                      )}
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="confirmPassword"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Confirm Password</FormLabel>
                      <FormControl>
                        <PasswordInput placeholder="••••••••" {...field} className="h-10" />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <Button
                  type="submit"
                  className="w-full h-10 font-medium"
                  disabled={form.formState.isSubmitting}
                >
                  {form.formState.isSubmitting ? 'Resetting...' : 'Reset password'}
                </Button>
              </form>
            </Form>
          </div>
        )}

        {status === 'success' && (
          <StatusCard
            icon={<ShieldCheck className="h-7 w-7 text-emerald-500" />}
            iconClass="bg-emerald-500/10"
            title="Password updated"
            description="Your password has been successfully updated. You can now log in with your new credentials."
            action={
              <Button asChild className="w-full h-10 font-medium mt-2">
                <Link to="/login">Go to login</Link>
              </Button>
            }
          />
        )}
      </div>
    </div>
  );
}
