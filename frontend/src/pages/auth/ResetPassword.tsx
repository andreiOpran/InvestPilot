import { useState } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { ShieldCheck, XCircle } from 'lucide-react';

import { resetPasswordSchema, type ResetPasswordFormValues } from '@/lib/schemas';
import { authApi } from '@/api/auth';
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

export function ResetPassword() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token');

  const [status, setStatus] = useState<'idle' | 'success' | 'invalid_token'>('idle');

  const form = useForm<ResetPasswordFormValues>({
    resolver: zodResolver(resetPasswordSchema),
    defaultValues: { token: token || '', newPassword: '', confirmPassword: '' },
  });

  const onSubmit = async (data: ResetPasswordFormValues) => {
    if (!token) {
      setStatus('invalid_token');
      return;
    }

    try {
      await authApi.resetPassword(data.token, data.newPassword);
      setStatus('success');
    } catch (error: any) {
      if (error.response?.status === 400 && error.response.data?.error?.toLowerCase().includes('token')) {
        setStatus('invalid_token');
      } else if (error.response?.status === 400 && error.response.data?.error) {
        form.setError('newPassword', {
          type: 'manual',
          message: error.response.data.error,
        });
      } else {
        form.setError('root', {
          type: 'manual',
          message: 'An unexpected error occurred. Please try again.',
        });
      }
    }
  };

  // If no token at all, just show invalid
  if (!token) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted/30 p-6">
        <Card className="w-full max-w-md shadow-xl border-border/50">
          <CardContent className="flex flex-col items-center space-y-6 pt-10 pb-8">
            <XCircle className="h-20 w-20 text-destructive" />
            <div className="space-y-2 text-center">
              <h3 className="text-xl font-semibold">Invalid Request</h3>
              <p className="text-muted-foreground text-sm">
                The password reset link is missing or invalid.
              </p>
            </div>
            <Button asChild className="w-full h-11 text-base">
              <Link to="/forgot-password">Request New Link</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/30 p-6">
      <Card className="w-full max-w-md shadow-xl border-border/50">
        
        {status === 'idle' && (
          <>
            <CardHeader className="text-center pb-2">
              <CardTitle className="text-3xl font-bold tracking-tight">Set New Password</CardTitle>
              <CardDescription>
                Please enter a strong password for your account.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6 pt-4">
              <Form {...form}>
                <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
                  {form.formState.errors.root && (
                    <div className="rounded-md bg-destructive/15 p-3">
                      <p className="text-sm font-medium text-destructive">
                        {form.formState.errors.root.message}
                      </p>
                    </div>
                  )}

                  <FormField
                    control={form.control}
                    name="newPassword"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>New Password</FormLabel>
                        <FormControl>
                          <Input type="password" placeholder="••••••••" {...field} className="h-11" />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="confirmPassword"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Confirm Password</FormLabel>
                        <FormControl>
                          <Input type="password" placeholder="••••••••" {...field} className="h-11" />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <Button
                    type="submit"
                    className="w-full h-11 text-base font-semibold mt-4"
                    disabled={form.formState.isSubmitting}
                  >
                    {form.formState.isSubmitting ? 'Resetting...' : 'Reset Password'}
                  </Button>
                </form>
              </Form>
            </CardContent>
          </>
        )}

        {status === 'success' && (
          <CardContent className="flex flex-col items-center space-y-6 pt-10 pb-8">
            <ShieldCheck className="h-20 w-20 text-emerald-500" />
            <div className="space-y-2 text-center">
              <h3 className="text-xl font-semibold">Password Reset</h3>
              <p className="text-muted-foreground text-sm">
                Your password has been successfully updated. You can now log in.
              </p>
            </div>
            <Button asChild className="w-full h-11 text-base">
              <Link to="/login">Go to Login</Link>
            </Button>
          </CardContent>
        )}

        {status === 'invalid_token' && (
          <CardContent className="flex flex-col items-center space-y-6 pt-10 pb-8">
            <XCircle className="h-20 w-20 text-destructive" />
            <div className="space-y-2 text-center">
              <h3 className="text-xl font-semibold">Link Expired</h3>
              <p className="text-muted-foreground text-sm">
                This password reset link is invalid or has expired.
              </p>
            </div>
            <Button asChild className="w-full h-11 text-base">
              <Link to="/forgot-password">Request New Link</Link>
            </Button>
          </CardContent>
        )}

      </Card>
    </div>
  );
}
