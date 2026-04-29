import { apiClient } from './client';

export const userApi = {
  getUser: () => 
    apiClient.get('/user'),
  
  updateProfile: (riskTolerance: number, investmentHorizon: number) => 
    apiClient.put('/user/profile', { risk_tolerance: riskTolerance, investment_horizon: investmentHorizon }),
  
  deposit: (amount: number) => 
    apiClient.post('/deposit', { amount }),
  
  cashout: (amount: number) => 
    apiClient.post('/cashout', { amount }),
  
  createDepositIntent: (amount: number) => 
    apiClient.post('/deposit/intent', { amount }),
};

export const onboardingApi = {
  getQuestions: () =>
    apiClient.get<{ questions: { id: string; text: string; options: { id: string; text: string }[] }[] }>('/onboarding/questions'),
  
  submitOnboarding: (answers: Record<string, string>) =>
    apiClient.post('/onboarding/submit', { answers }),
};

