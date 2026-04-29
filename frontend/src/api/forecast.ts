import { apiClient } from './client';

export const forecastApi = {
  requestForecast: (initialInvestment: number, monthlyContribution: number, years: number) => 
    apiClient.post('/forecast', { 
      initial_investment: initialInvestment, 
      monthly_contribution: monthlyContribution, 
      years: years 
    }),
  
  getForecastStatus: (taskId: string) => 
    apiClient.get(`/forecast/status/${taskId}`),
};
