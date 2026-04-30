import { z } from "zod";
import zxcvbn from "zxcvbn";


// Reusable Base Schemas

// email format validation
const emailValidation = z
  .string()
  .min(1, "Email is required")
  .regex(/^[^\s@]+@[^\s@]+\.[^\s@]+$/, "Please enter a valid email address");

// match go password complexity requirements
const passwordComplexity = z
  .string()
  .min(10, "Password must be at least 10 characters long")
  .max(128, "Password must be at most 128 characters long")
  .regex(/[A-Z]/, "Password must contain at least one uppercase letter")
  .regex(/[a-z]/, "Password must contain at least one lowercase letter")
  .regex(/[0-9]/, "Password must contain at least one number")
  .regex(/[\W_]/, "Password must contain at least one special character");

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

export const registerSchema = z.object({
  email: emailValidation,
  password: passwordComplexity,
}).refine((data) => {
  if (!data.password || !data.email) return true;
  const result = zxcvbn(data.password, [data.email]);
  return result.score >= 3;
}, {
  message: "Password is too weak, easily guessable, or commonly used. Please make it stronger.",
  path: ["password"],
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
    newPassword: passwordComplexity,
    confirmPassword: z.string().min(1, "Please confirm your password"),
  })
  .refine((data) => data.newPassword === data.confirmPassword, {
    message: "Passwords do not match",
    path: ["confirmPassword"],
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
export type ForecastFormValues = z.infer<typeof forecastSchema>;
