import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { toast } from 'sonner';
import { Loader2, ChevronRight, ChevronLeft, TrendingUp, Clock, CheckCircle2 } from 'lucide-react';

import { onboardingApi, userApi } from '@/api/user';
import { useAuthStore } from '@/stores/authStore';

import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
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
      .catch(() => {
        setViewState('error');
      });
  }, []);

  const currentQuestion = questions[currentStep];
  const totalSteps = questions.length;
  const progress = totalSteps > 0 ? ((currentStep + 1) / totalSteps) * 100 : 0;
  const isLastStep = currentStep === totalSteps - 1;
  const currentAnswer = answers[currentQuestion?.id];

  const handleNext = () => {
    if (currentStep < totalSteps - 1) {
      setCurrentStep((s) => s + 1);
    }
  };

  const handleBack = () => {
    if (currentStep > 0) {
      setCurrentStep((s) => s - 1);
    }
  };

  const handleSubmit = async () => {
    setViewState('submitting');
    try {
      await onboardingApi.submitOnboarding(answers);

      // Fetch updated user profile to get the computed risk/horizon values
      const userRes = await userApi.getUser();
      const updatedUser = userRes.data;
      setUser(updatedUser);

      setSummaryData({
        riskTolerance: updatedUser.risk_tolerance,
        investmentHorizon: updatedUser.investment_horizon,
      });
      setViewState('summary');
    } catch (error: any) {
      toast.error(error.response?.data?.error ?? 'Something went wrong. Please try again.');
      setViewState('questions'); // return user to final step with answers preserved
    }
  };

  // ─── Loading ────────────────────────────────────────────────────
  if (viewState === 'loading') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted/30 p-6">
        <Card className="w-full max-w-xl shadow-xl">
          <CardHeader>
            <Skeleton className="h-2 w-full rounded-full mb-4" />
            <Skeleton className="h-7 w-3/4" />
            <Skeleton className="h-4 w-1/2 mt-1" />
          </CardHeader>
          <CardContent className="space-y-3">
            {[...Array(4)].map((_, i) => (
              <Skeleton key={i} className="h-14 w-full rounded-lg" />
            ))}
          </CardContent>
        </Card>
      </div>
    );
  }

  // ─── Error ──────────────────────────────────────────────────────
  if (viewState === 'error') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted/30 p-6">
        <Card className="w-full max-w-xl shadow-xl text-center">
          <CardContent className="pt-10 pb-8 space-y-4">
            <p className="text-muted-foreground">Failed to load questionnaire. Please refresh and try again.</p>
            <Button onClick={() => window.location.reload()}>Try Again</Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  // ─── Summary Screen ─────────────────────────────────────────────
  if (viewState === 'summary' && summaryData) {
    const riskInfo = riskDescriptions[summaryData.riskTolerance] ?? riskDescriptions[3];
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted/30 p-6">
        <Card className="w-full max-w-xl shadow-xl">
          <CardHeader className="text-center pb-2">
            <div className="flex justify-center mb-4">
              <CheckCircle2 className="h-16 w-16 text-emerald-500" />
            </div>
            <CardTitle className="text-2xl font-bold">Your Profile is Ready!</CardTitle>
            <CardDescription>
              Based on your answers, here is your personalized investment profile.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6 pt-4">
            {/* Risk Tolerance Card */}
            <div className="rounded-xl border border-border/60 bg-muted/30 p-5 space-y-2">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                  <TrendingUp className="h-4 w-4" />
                  Risk Tolerance
                </div>
                <Badge variant="outline" className={`font-semibold text-sm ${riskInfo.color}`}>
                  Level {summaryData.riskTolerance} / 5
                </Badge>
              </div>
              <p className="text-xl font-bold">{riskInfo.label}</p>
              <p className="text-sm text-muted-foreground leading-relaxed">{riskInfo.description}</p>
            </div>

            {/* Investment Horizon Card */}
            <div className="rounded-xl border border-border/60 bg-muted/30 p-5 space-y-2">
              <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                <Clock className="h-4 w-4" />
                Investment Horizon
              </div>
              <p className="text-xl font-bold">{summaryData.investmentHorizon} years</p>
              <p className="text-sm text-muted-foreground">
                Your portfolio will be structured and rebalanced with a {summaryData.investmentHorizon}-year timeframe in mind.
              </p>
            </div>

            <Button
              className="w-full h-11 text-base font-semibold"
              onClick={() => navigate(isEditMode ? '/settings' : '/dashboard')}
            >
              {isEditMode ? 'Return to Settings' : 'Go to Dashboard'}
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  // ─── Questionnaire ──────────────────────────────────────────────
  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/30 p-6">
      <div className="w-full max-w-xl space-y-6">
        {/* Header */}
        <div className="text-center space-y-1">
          <p className="text-sm text-muted-foreground font-medium">
            Question {currentStep + 1} of {totalSteps}
          </p>
          <Progress value={progress} className="h-2" />
        </div>

        <Card className="shadow-xl border-border/50">
          <CardHeader className="pb-2">
            <CardTitle className="text-xl font-semibold leading-snug">
              {currentQuestion?.text}
            </CardTitle>
          </CardHeader>

          <CardContent className="space-y-6 pt-2">
            {/* Options */}
            <RadioGroup
              value={currentAnswer ?? ''}
              onValueChange={(value) =>
                setAnswers((prev) => ({ ...prev, [currentQuestion.id]: value }))
              }
              className="space-y-3"
            >
              {currentQuestion?.options.map((option) => {
                const isSelected = currentAnswer === option.id;
                return (
                  <label
                    key={option.id}
                    htmlFor={option.id}
                    className={`flex items-center gap-4 rounded-xl border p-4 cursor-pointer transition-all duration-150
                      ${isSelected
                        ? 'border-primary bg-primary/5 shadow-sm'
                        : 'border-border/60 hover:border-primary/40 hover:bg-muted/50'
                      }`}
                  >
                    <RadioGroupItem value={option.id} id={option.id} className="shrink-0" />
                    <Label htmlFor={option.id} className="cursor-pointer text-sm font-medium leading-snug">
                      {option.text}
                    </Label>
                  </label>
                );
              })}
            </RadioGroup>

            {/* Navigation Buttons */}
            <div className="flex items-center justify-between pt-2">
              <Button
                variant="ghost"
                onClick={handleBack}
                disabled={currentStep === 0 || viewState === 'submitting'}
                className="gap-1"
              >
                <ChevronLeft className="h-4 w-4" />
                Back
              </Button>

              {isLastStep ? (
                <Button
                  onClick={handleSubmit}
                  disabled={!currentAnswer || viewState === 'submitting'}
                  className="h-10 px-6 font-semibold gap-2"
                >
                  {viewState === 'submitting' ? (
                    <>
                      <Loader2 className="h-4 w-4 animate-spin" />
                      Calculating...
                    </>
                  ) : (
                    'Calculate My Profile'
                  )}
                </Button>
              ) : (
                <Button
                  onClick={handleNext}
                  disabled={!currentAnswer}
                  className="h-10 px-6 font-semibold gap-1"
                >
                  Next
                  <ChevronRight className="h-4 w-4" />
                </Button>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Progress dots */}
        <div className="flex justify-center gap-2">
          {questions.map((_, i) => (
            <div
              key={i}
              className={`h-2 rounded-full transition-all duration-300 ${
                i === currentStep
                  ? 'w-6 bg-primary'
                  : i < currentStep
                    ? 'w-2 bg-primary/50'
                    : 'w-2 bg-border'
              }`}
            />
          ))}
        </div>
      </div>
    </div>
  );
}
