import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { toast } from 'sonner';
import { Loader2, ChevronRight, ChevronLeft, TrendingUp, Clock, CheckCircle2, Landmark } from 'lucide-react';

import { onboardingApi, userApi } from '@/api/user';
import { useAuthStore } from '@/stores/authStore';

import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';

interface OnboardingOption {
  id: string;
  text: string;
}

interface OnboardingQuestion {
  id: string;
  text: string;
  options: OnboardingOption[];
}

type ViewState = 'loading' | 'error' | 'questions' | 'submitting' | 'summary';

const riskDescriptions: Record<number, { label: string; description: string; color: string }> = {
  1: {
    label: 'Very Conservative',
    description: 'Your portfolio will be mostly bonds and stable assets, prioritizing capital preservation over growth. Ideal for short-term goals or low risk appetite.',
    color: 'text-blue-500',
  },
  2: {
    label: 'Conservative',
    description: 'A modest allocation toward growth assets with a heavy emphasis on stability. Suitable for investors who want slow, steady growth with minimal volatility.',
    color: 'text-cyan-500',
  },
  3: {
    label: 'Balanced',
    description: 'A balanced mix of equities and fixed income. You accept moderate short-term fluctuations in exchange for solid long-term returns.',
    color: 'text-emerald-500',
  },
  4: {
    label: 'Growth',
    description: 'A growth-oriented portfolio with a high allocation to equities. You are comfortable with significant short-term swings to chase higher long-term gains.',
    color: 'text-amber-500',
  },
  5: {
    label: 'Aggressive Growth',
    description: 'Maximum exposure to high-growth assets. You are prepared for high volatility in pursuit of the highest possible long-term returns.',
    color: 'text-rose-500',
  },
};

export function Onboarding() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const isEditMode = searchParams.get('edit') === 'true';
  const { setUser } = useAuthStore();

  const [viewState, setViewState] = useState<ViewState>('loading');
  const [questions, setQuestions] = useState<OnboardingQuestion[]>([]);
  const [currentStep, setCurrentStep] = useState(0);
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [summaryData, setSummaryData] = useState<{ riskTolerance: number; investmentHorizon: number } | null>(null);

  useEffect(() => {
    onboardingApi.getQuestions()
      .then((res) => {
        setQuestions(res.data.questions);
        setViewState('questions');
      })
      .catch(() => setViewState('error'));
  }, []);

  const currentQuestion = questions[currentStep];
  const totalSteps = questions.length;
  const progress = totalSteps > 0 ? ((currentStep + 1) / totalSteps) * 100 : 0;
  const isLastStep = currentStep === totalSteps - 1;
  const currentAnswer = answers[currentQuestion?.id];

  const handleNext = () => { if (currentStep < totalSteps - 1) setCurrentStep((s) => s + 1); };
  const handleBack = () => { if (currentStep > 0) setCurrentStep((s) => s - 1); };

  const handleSubmit = async () => {
    setViewState('submitting');
    try {
      await onboardingApi.submitOnboarding(answers);
      const userRes = await userApi.getUser();
      const updatedUser = userRes.data;
      setUser(updatedUser);
      setSummaryData({ riskTolerance: updatedUser.risk_tolerance, investmentHorizon: updatedUser.investment_horizon });
      setViewState('summary');
    } catch (error: any) {
      toast.error(error.response?.data?.error ?? 'Something went wrong. Please try again.');
      setViewState('questions');
    }
  };

  // Loading
  if (viewState === 'loading') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted/30 px-4 py-12">
        <div className="w-full max-w-lg space-y-6">
          <div className="flex flex-col items-center gap-2">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl border bg-background shadow-sm">
              <Landmark className="h-5 w-5 text-primary" />
            </div>
          </div>
          <div className="rounded-xl border bg-card shadow-sm p-6 space-y-4">
            <Skeleton className="h-2 w-full rounded-full" />
            <Skeleton className="h-5 w-3/4" />
            <Skeleton className="h-4 w-1/2" />
            {[...Array(4)].map((_, i) => <Skeleton key={i} className="h-12 w-full rounded-lg" />)}
          </div>
        </div>
      </div>
    );
  }

  // Error
  if (viewState === 'error') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted/30 px-4 py-12">
        <div className="w-full max-w-sm space-y-6">
          <div className="flex flex-col items-center gap-2 text-center">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl border bg-background shadow-sm">
              <Landmark className="h-5 w-5 text-primary" />
            </div>
          </div>
          <div className="rounded-xl border bg-card shadow-sm p-8 text-center space-y-4">
            <p className="text-sm text-muted-foreground">Failed to load questionnaire. Please refresh and try again.</p>
            <Button onClick={() => window.location.reload()} className="h-9 px-6 text-sm font-medium">
              Try again
            </Button>
          </div>
        </div>
      </div>
    );
  }

  // Summary
  if (viewState === 'summary' && summaryData) {
    const riskInfo = riskDescriptions[summaryData.riskTolerance] ?? riskDescriptions[3];
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted/30 px-4 py-12">
        <div className="w-full max-w-sm space-y-6">
          <div className="flex flex-col items-center gap-2 text-center">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl border bg-background shadow-sm">
              <Landmark className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="text-sm font-semibold tracking-tight">Profile ready</p>
              <p className="text-xs text-muted-foreground mt-0.5">Your personalized investment profile</p>
            </div>
          </div>

          <div className="rounded-xl border bg-card shadow-sm p-6 space-y-4">
            <div className="flex flex-col items-center text-center pb-2">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-emerald-500/10 mb-3">
                <CheckCircle2 className="h-6 w-6 text-emerald-500" />
              </div>
              <p className="font-semibold tracking-tight">Your profile is ready</p>
              <p className="text-xs text-muted-foreground mt-1">Based on your answers, here is your personalized investment profile.</p>
            </div>

            <div className="space-y-3">
              <div className="rounded-lg border bg-muted/30 p-4 space-y-1.5">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <TrendingUp className="h-3.5 w-3.5" />
                    Risk Tolerance
                  </div>
                  <Badge variant="outline" className={`text-xs font-semibold ${riskInfo.color}`}>
                    Level {summaryData.riskTolerance}/5
                  </Badge>
                </div>
                <p className="font-semibold text-sm">{riskInfo.label}</p>
                <p className="text-xs text-muted-foreground leading-relaxed">{riskInfo.description}</p>
              </div>

              <div className="rounded-lg border bg-muted/30 p-4 space-y-1.5">
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <Clock className="h-3.5 w-3.5" />
                  Investment Horizon
                </div>
                <p className="font-semibold text-sm">{summaryData.investmentHorizon} years</p>
                <p className="text-xs text-muted-foreground leading-relaxed">
                  Your portfolio will be structured with a {summaryData.investmentHorizon}-year timeframe in mind.
                </p>
              </div>
            </div>

            <Button
              className="w-full h-9 font-medium text-sm"
              onClick={() => navigate(isEditMode ? '/settings' : '/dashboard')}
            >
              {isEditMode ? 'Return to settings' : 'Go to dashboard'}
            </Button>
          </div>
        </div>
      </div>
    );
  }

  // Questionnaire
  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/30 px-4 py-12">
      <div className="w-full max-w-lg space-y-5">

        {/* Logo + progress */}
        <div className="flex flex-col items-center gap-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl border bg-background shadow-sm">
            <Landmark className="h-5 w-5 text-primary" />
          </div>
          <div className="w-full space-y-1.5">
            <div className="flex items-center justify-between text-xs text-muted-foreground">
              <span>Investment profile questionnaire</span>
              <span className="font-mono font-medium">
                {totalSteps > 0 ? `${currentStep + 1} / ${totalSteps}` : ''}
              </span>
            </div>
            <Progress value={progress} className="h-1.5" />
          </div>
        </div>

        {/* Card */}
        <div className="rounded-xl border bg-card shadow-sm p-6 space-y-5">
          <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
              Question {currentStep + 1}
            </p>
            <p className="font-semibold text-base leading-snug">{currentQuestion?.text}</p>
          </div>

          <RadioGroup
            value={currentAnswer ?? ''}
            onValueChange={(value) => {
              if (currentQuestion) {
                setAnswers((prev) => ({ ...prev, [currentQuestion.id]: value }));
              }
            }}
            className="space-y-2"
          >
            {currentQuestion?.options?.map((option) => {
              const isSelected = currentAnswer === option.id;
              return (
                <label
                  key={option.id}
                  htmlFor={option.id}
                  className={`flex items-center gap-3 rounded-lg border px-4 py-3 cursor-pointer transition-all duration-150 ${
                    isSelected
                      ? 'border-primary bg-primary/5'
                      : 'border-border/60 hover:border-primary/30 hover:bg-muted/40'
                  }`}
                >
                  <RadioGroupItem value={option.id} id={option.id} className="shrink-0" />
                  <Label htmlFor={option.id} className="cursor-pointer text-sm leading-snug font-normal">
                    {option.text}
                  </Label>
                </label>
              );
            })}
          </RadioGroup>

          <div className="flex items-center justify-between pt-1">
            <Button
              variant="ghost"
              onClick={handleBack}
              disabled={currentStep === 0 || viewState === 'submitting'}
              className="gap-1 text-sm h-9"
            >
              <ChevronLeft className="h-4 w-4" />
              Back
            </Button>

            {isLastStep ? (
              <Button
                onClick={handleSubmit}
                disabled={!currentAnswer || viewState === 'submitting'}
                className="h-9 px-6 text-sm font-medium gap-2"
              >
                {viewState === 'submitting' ? (
                  <>
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    Calculating...
                  </>
                ) : (
                  'Calculate my profile'
                )}
              </Button>
            ) : (
              <Button
                onClick={handleNext}
                disabled={!currentAnswer}
                className="h-9 px-6 text-sm font-medium gap-1"
              >
                Next
                <ChevronRight className="h-4 w-4" />
              </Button>
            )}
          </div>
        </div>

        {/* Step dots */}
        <div className="flex justify-center gap-1.5">
          {questions.map((_, i) => (
            <div
              key={i}
              className={`h-1.5 rounded-full transition-all duration-300 ${
                i === currentStep
                  ? 'w-5 bg-primary'
                  : i < currentStep
                    ? 'w-1.5 bg-primary/40'
                    : 'w-1.5 bg-border'
              }`}
            />
          ))}
        </div>
      </div>
    </div>
  );
}
