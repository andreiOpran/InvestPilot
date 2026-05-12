import { z } from "zod";
import zxcvbn from "zxcvbn";


// Reusable Base Schemas

// email format validation
const emailValidation = z
  .string()
  .min(1, "Email is required")
  .regex(/^[^\s@]+@[^\s@]+\.[^\s@]+$/, "Please enter a valid email address");

// Validates password requirements one at a time (early exit) so only the first
// unmet requirement is shown. userInputs are fed to zxcvbn for context-aware scoring.
// Validates password requirements one at a time (early exit) so only the first
// unmet requirement is shown. userInputs are fed to zxcvbn for context-aware scoring.
function addPasswordIssues(
  password: string,
  userInputs: string[],
  ctx: z.RefinementCtx,
  path: string[]
) {
  if (password.length < 10) {
    ctx.addIssue({ code: "custom", message: "Password must be at least 10 characters", path });
    return;
  }
  if (password.length > 128) {
    ctx.addIssue({ code: "custom", message: "Password must be at most 128 characters", path });
    return;
  }
  if (!/[A-Z]/.test(password)) {
    ctx.addIssue({ code: "custom", message: "Password must contain at least one uppercase letter", path });
    return;
  }
  if (!/[a-z]/.test(password)) {
    ctx.addIssue({ code: "custom", message: "Password must contain at least one lowercase letter", path });
    return;
  }
  if (!/[0-9]/.test(password)) {
    ctx.addIssue({ code: "custom", message: "Password must contain at least one number", path });
    return;
  }
  if (!/[\W_]/.test(password)) {
    ctx.addIssue({ code: "custom", message: "Must contain at least one special character (e.g. !@#$%)", path });
    return;
  }
  if (zxcvbn(password, userInputs).score < 3) {
    ctx.addIssue({ code: "custom", message: "Password is too weak or easily guessable — add more variety", path });
  }
}

// 6-digit numeric token for 2FA
const token2FASchema = z
  .string()
  .length(6, "Token must be exactly 6 characters")
  .regex(/^\d{6}$/, "Token must contain only numbers");

// shared positive amount for financial transactions
const amountSchema = z.coerce
  .number({ message: "Amount must be a valid number" })
  .positive("Amount must be greater than 0");


// Authentication Schemas

export const registerSchema = z
  .object({
    email: emailValidation,
    password: z.string().min(1, "Password is required"),
    confirmPassword: z.string().min(1, "Please confirm your password"),
  })
  .superRefine(({ email, password, confirmPassword }, ctx) => {
    if (password) addPasswordIssues(password, [email], ctx, ["password"]);
    if (confirmPassword && password !== confirmPassword) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: "Passwords do not match", path: ["confirmPassword"] });
    }
  });

export const loginSchema = z.object({
  email: emailValidation,
  password: z.string().min(1, "Password is required"),
});

export const verify2FASchema = z.object({
  email: emailValidation,
  password: z.string().min(1, "Password is required"),
  token: token2FASchema,
});

export const enable2FASchema = z.object({
  token: token2FASchema,
});

export const forgotPasswordSchema = z.object({
  email: emailValidation,
});

export const resetPasswordSchema = z
  .object({
    token: z.string().min(1, "Reset token is required"),
    newPassword: z.string().min(1, "Password is required"),
    confirmPassword: z.string().min(1, "Please confirm your password"),
  })
  .superRefine(({ newPassword, confirmPassword }, ctx) => {
    if (newPassword) {
      addPasswordIssues(newPassword, [], ctx, ["newPassword"]);
    }
    if (newPassword !== confirmPassword) {
      ctx.addIssue({ code: "custom", message: "Passwords do not match", path: ["confirmPassword"] });
    }
  });


// Profile and Onboarding Schemas

export const onboardingSchema = z.object({
  // validates the map[string]string structure from backend OnboardingSubmitRequest
  answers: z.record(
    z.string(),
    z.string().min(1, "Please select an answer for this question")
  ).refine(record => Object.keys(record).length > 0, "You must answer the questions"),
});

// Based on UpdateProfileRequest
export const updateProfileSchema = z.object({
  risk_tolerance: z.coerce.number().int().min(1).max(5),
  investment_horizon: z.coerce.number().int().min(1).max(50),
});


// Financial Transaction Schemas

export const depositSchema = z.object({
  amount: amountSchema,
});

export const cashoutSchema = z.object({
  amount: amountSchema,
});

export const investSchema = z.object({
  amount: amountSchema,
});

export const sellSchema = z.object({
  amount: amountSchema,
});


// Analysis and Tools Schemas

export const forecastSchema = z.object({
  initial_investment: z.coerce
    .number({ message: "Must be a valid number" })
    .min(0, "Initial investment cannot be negative"),
  monthly_contribution: z.coerce
    .number({ message: "Must be a valid number" })
    .min(0, "Contribution cannot be negative")
    .optional()
    .default(0),
  years: z.coerce
    .number({ message: "Please enter a valid number of years" })
    .int("Years must be a whole number")
    .min(1, "Must forecast at least 1 year")
    .max(50, "Cannot forecast beyond 50 years"),
});

// infer types from schemas to be used across components
export type RegisterFormValues = z.infer<typeof registerSchema>;
export type LoginFormValues = z.infer<typeof loginSchema>;
export type Verify2FAFormValues = z.infer<typeof verify2FASchema>;
export type Enable2FAFormValues = z.infer<typeof enable2FASchema>;
export type ForgotPasswordFormValues = z.infer<typeof forgotPasswordSchema>;
export type ResetPasswordFormValues = z.infer<typeof resetPasswordSchema>;
export type OnboardingFormValues = z.infer<typeof onboardingSchema>;
export type UpdateProfileFormValues = z.infer<typeof updateProfileSchema>;
export type DepositFormValues = z.infer<typeof depositSchema>;
export type CashoutFormValues = z.infer<typeof cashoutSchema>;
export type InvestFormValues = z.infer<typeof investSchema>;
export type SellFormValues = z.infer<typeof sellSchema>;
export type ForecastFormValues = z.infer<typeof forecastSchema>;
