import { apiClient } from './client';

export const portfolioApi = {
  invest: (amount: number) => 
    apiClient.post('/invest', { amount }),
  
  getHistory: (range: string) => 
    apiClient.get(`/portfolio/history?range=${range}`),
  
  getPortfolio: () => 
    apiClient.get('/portfolio'),
  
  getTransactions: (page?: number, limit?: number) => 
    apiClient.get('/transactions', { params: { page, limit } }),
};
