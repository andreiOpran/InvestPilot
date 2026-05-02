import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Turnstile } from '@marsidev/react-turnstile';
import { toast } from 'sonner';
import { MailCheck, ArrowLeft, Landmark } from 'lucide-react';

import { forgotPasswordSchema, type ForgotPasswordFormValues } from '@/lib/schemas';
import { authApi } from '@/api/auth';
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

export function ForgotPassword() {
  const [turnstileToken, setTurnstileToken] = useState<string | null>(null);
  const [isSubmitted, setIsSubmitted] = useState(false);

  const form = useForm<ForgotPasswordFormValues>({
    resolver: zodResolver(forgotPasswordSchema),
    defaultValues: { email: '' },
  });

  const onSubmit = async (data: ForgotPasswordFormValues) => {
    if (!turnstileToken) {
      toast.error('Anti-bot check failed. Please wait for the Turnstile challenge.');
      return;
    }

    try {
      await authApi.forgotPassword(data.email, turnstileToken);
      setIsSubmitted(true);
    } catch (error) {
      toast.error('An error occurred. Please try again later.');
    }
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
            <p className="text-sm font-semibold tracking-tight">InvestPilot</p>
            <p className="text-xs text-muted-foreground mt-0.5">
              {isSubmitted ? 'Check your inbox' : 'Reset your password'}
            </p>
          </div>
        </div>

        {/* Card */}
        <div className="rounded-xl border bg-card shadow-sm p-6 space-y-5">

          {!isSubmitted ? (
            <>
              <div className="text-center space-y-1">
                <p className="text-sm text-muted-foreground">
                  Enter your email address and we&apos;ll send you a link to reset your password.
                </p>
              </div>

              <Form {...form}>
                <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
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

                  <div className="flex justify-center">
                    <Turnstile
                      siteKey={import.meta.env.VITE_TURNSTILE_SITE_KEY || '3x00000000000000000000FF'}
                      onSuccess={(token) => setTurnstileToken(token)}
                      onError={() => toast.error('Anti-bot check failed. Please try again.')}
                      options={{ theme: 'auto' }}
                    />
                  </div>

                  <Button
                    type="submit"
                    className="w-full h-10 font-medium"
                    disabled={form.formState.isSubmitting || !turnstileToken}
                  >
                    {form.formState.isSubmitting ? 'Sending...' : 'Send reset link'}
                  </Button>
                </form>
              </Form>

              <Button variant="ghost" asChild className="w-full text-muted-foreground text-sm">
                <Link to="/login">
                  <ArrowLeft className="mr-2 h-3.5 w-3.5" />
                  Back to login
                </Link>
              </Button>
            </>
          ) : (
            <div className="flex flex-col items-center text-center space-y-4 py-4">
              <div className="flex h-14 w-14 items-center justify-center rounded-full bg-primary/10">
                <MailCheck className="h-7 w-7 text-primary" />
              </div>
              <div className="space-y-1">
                <p className="font-semibold text-sm">Email sent</p>
                <p className="text-xs text-muted-foreground leading-relaxed">
                  If an account with that email exists, a reset link has been sent.
                  Please check your inbox and click the link to reset your password.
                </p>
              </div>
              <Button asChild variant="outline" className="w-full h-10 font-medium">
                <Link to="/login">Return to login</Link>
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
