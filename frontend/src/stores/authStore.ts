import { create } from 'zustand';

export interface User {
  user_id: number;
  email: string;
  risk_tolerance: number;
  investment_horizon: number;
  wallet_balance: number;
}

interface AuthState {
  accessToken: string | null;
  user: User | null;
  status: 'loading' | 'authenticated' | 'unauthenticated';
  securityAlert: boolean;
  setAccessToken: (token: string) => void;
  setUser: (user: User) => void;
  setStatus: (status: 'loading' | 'authenticated' | 'unauthenticated') => void;
  setSecurityAlert: (alert: boolean) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  accessToken: null,
  user: null,
  status: 'loading',
  securityAlert: false,
  setAccessToken: (token) => set({ accessToken: token }),
  setUser: (user) => set({ user }),
  setStatus: (status) => set({ status }),
  setSecurityAlert: (securityAlert) => set({ securityAlert }),
  clearAuth: () => set({ accessToken: null, user: null, status: 'unauthenticated' }),
}));
