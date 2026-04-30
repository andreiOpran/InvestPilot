import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Turnstile } from '@marsidev/react-turnstile';
import { toast } from 'sonner';
import { Landmark } from 'lucide-react';

import { registerSchema, type RegisterFormValues } from '@/lib/schemas';
import { authApi } from '@/api/auth';
import { Alert, AlertDescription } from '@/components/ui/alert';
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

export function Register() {
  const navigate = useNavigate();
  const [turnstileToken, setTurnstileToken] = useState<string | null>(null);

  const form = useForm<RegisterFormValues>({
    resolver: zodResolver(registerSchema),
    mode: 'onChange',
    defaultValues: { email: '', password: '' },
  });

  const onSubmit = async (data: RegisterFormValues) => {
    if (!turnstileToken) {
      toast.error('Anti-bot check failed. Please wait for the Turnstile challenge.');
      return;
    }

    try {
      await authApi.register(data.email, data.password, turnstileToken);
      navigate('/register-success');
    } catch (error: any) {
      if (error.response?.status === 409) {
        form.setError('email', { type: 'manual', message: 'An account with this email already exists.' });
      } else if (error.response?.status === 400 && error.response.data?.error) {
        form.setError('password', {
          type: 'manual',
          message: error.response.data.error.charAt(0).toUpperCase() + error.response.data.error.slice(1) + '.',
        });
      } else {
        toast.error('Registration failed. Please try again.');
      }
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
            <p className="text-sm font-semibold tracking-tight">RoboAdvisor</p>
            <p className="text-xs text-muted-foreground mt-0.5">Create your account</p>
          </div>
        </div>

        {/* Card */}
        <div className="rounded-xl border bg-card shadow-sm p-6 space-y-5">
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">

              {(form.formState.errors.email || form.formState.errors.password) && (
                <Alert variant="destructive">
                  <AlertDescription className="space-y-1">
                    {form.formState.errors.email?.message && <p>{form.formState.errors.email.message}</p>}
                    {form.formState.errors.password?.message && <p>{form.formState.errors.password.message}</p>}
                  </AlertDescription>
                </Alert>
              )}

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
                    <FormLabel className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Password</FormLabel>
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
                disabled={form.formState.isSubmitting || !turnstileToken}
              >
                {form.formState.isSubmitting ? 'Creating account...' : 'Create account'}
              </Button>
            </form>
          </Form>
        </div>

        <p className="text-center text-xs text-muted-foreground">
          Already have an account?{' '}
          <Link to="/login" className="font-medium text-foreground hover:text-primary transition-colors">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  );
}
