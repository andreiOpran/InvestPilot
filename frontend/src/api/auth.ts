import { apiClient } from './client';

export const authApi = {
  register: (email: string, password: string, turnstileToken: string) => 
    apiClient.post('/register', { email, password, turnstile_token: turnstileToken }),
  
  verifyEmail: (token: string) => 
    apiClient.get(`/verify-email?token=${token}`),
  
  login: (email: string, password: string, turnstileToken: string) => 
    apiClient.post('/login', { email, password, turnstile_token: turnstileToken }),
  
  verify2FA: (email: string, password: string, totpToken: string) => 
    apiClient.post('/verify-2fa', { email, password, token: totpToken }),
  
  logout: () => 
    apiClient.post('/logout'),
  
  refreshToken: () => 
    apiClient.post('/refresh-token'),
  
  forgotPassword: (email: string, turnstileToken: string) => 
    apiClient.post('/forgot-password', { email, turnstile_token: turnstileToken }),
  
  resetPassword: (token: string, newPassword: string) => 
    apiClient.post('/reset-password', { token, new_password: newPassword }),
  
  setup2FA: () => 
    apiClient.get('/2fa/setup'),
  
  enable2FA: (token: string) => 
    apiClient.post('/2fa/enable', { token }),
};
