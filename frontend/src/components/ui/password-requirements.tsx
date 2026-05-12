import { Check, X } from 'lucide-react';
import { cn } from '@/lib/utils';

interface Requirement {
  label: string;
  test: (p: string) => boolean;
}

const REQUIREMENTS: Requirement[] = [
  { label: 'At least 10 characters',     test: (p) => p.length >= 10 },
  { label: 'Uppercase letter',           test: (p) => /[A-Z]/.test(p) },
  { label: 'Lowercase letter',           test: (p) => /[a-z]/.test(p) },
  { label: 'Number',                     test: (p) => /[0-9]/.test(p) },
  { label: 'Special character (!@#$…)',  test: (p) => /[\W_]/.test(p) },
];

interface PasswordRequirementsProps {
  password: string;
}

export function PasswordRequirements({ password }: PasswordRequirementsProps) {
  return (
    <div className="animate-in fade-in slide-in-from-top-1 duration-150 rounded-lg border bg-muted/40 px-3 py-2.5 mt-1 space-y-1.5">
      {REQUIREMENTS.map(({ label, test }) => {
        const met = password.length > 0 && test(password);
        return (
          <div
            key={label}
            className={cn(
              'flex items-center gap-2 text-xs transition-colors duration-150',
              password.length === 0
                ? 'text-muted-foreground'
                : met
                ? 'text-emerald-600 dark:text-emerald-500'
                : 'text-destructive',
            )}
          >
            {met ? (
              <Check className="h-3 w-3 shrink-0" />
            ) : (
              <X className="h-3 w-3 shrink-0 opacity-70" />
            )}
            <span>{label}</span>
          </div>
        );
      })}
    </div>
  );
}
