import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Turnstile } from '@marsidev/react-turnstile';
import { toast } from 'sonner';
import { MailCheck, ArrowLeft } from 'lucide-react';

import { forgotPasswordSchema, type ForgotPasswordFormValues } from '@/lib/schemas';
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
      // Always show success regardless of the response, to prevent email enumeration.
      setIsSubmitted(true);
    } catch (error) {
      // In case of 500 or rate limits, still show generic error,
      // but otherwise pretend it worked for 4xx (except 429).
      toast.error('An error occurred. Please try again later.');
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/30 p-6">
      <Card className="w-full max-w-md shadow-xl border-border/50">
        {!isSubmitted ? (
          <>
            <CardHeader className="text-center pb-2">
              <CardTitle className="text-3xl font-bold tracking-tight">Forgot Password</CardTitle>
              <CardDescription>
                Enter your email address and we&apos;ll send you a link to reset your password.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6 pt-4">
              <Form {...form}>
                <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
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
                    disabled={form.formState.isSubmitting || !turnstileToken}
                  >
                    {form.formState.isSubmitting ? 'Sending Link...' : 'Send Reset Link'}
                  </Button>
                </form>
              </Form>

              <div className="text-center">
                <Button variant="ghost" asChild className="text-muted-foreground">
                  <Link to="/login">
                    <ArrowLeft className="mr-2 h-4 w-4" />
                    Back to login
                  </Link>
                </Button>
              </div>
            </CardContent>
          </>
        ) : (
          <>
            <CardHeader className="text-center pb-2">
              <div className="flex justify-center mb-4">
                <MailCheck className="h-20 w-20 text-primary" />
              </div>
              <CardTitle className="text-2xl font-bold tracking-tight">Check your inbox</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col items-center space-y-6 pt-4 pb-8">
              <p className="text-center text-muted-foreground text-sm">
                If an account with that email exists, a reset link has been sent.
                Please check your inbox and click the link to reset your password.
              </p>
              <Button asChild variant="outline" className="w-full h-11 text-base">
                <Link to="/login">Return to Login</Link>
              </Button>
            </CardContent>
          </>
        )}
      </Card>
    </div>
  );
}
