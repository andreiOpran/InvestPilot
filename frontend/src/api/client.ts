import axios from 'axios';
import { useAuthStore } from '@/stores/authStore';
import { toast } from 'sonner';

export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  withCredentials: true,
});

let isRefreshing = false;
let requestQueue: Array<{ resolve: (token: string) => void; reject: (err: any) => void }> = [];

apiClient.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken;
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    if (error.response && error.response.status >= 500) {
      toast.error('Server error. Please try again shortly.');
      return Promise.reject(error);
    }

    if (error.response?.status === 429) {
      toast.error('Too many requests. Please slow down.');
      return Promise.reject(error);
    }

    if (
      error.response?.status === 423 ||
      (error.response?.status === 403 && error.response.data?.message?.toLowerCase().includes('lock'))
    ) {
      toast.error('Account temporarily locked. Try again in 15 minutes.');
      return Promise.reject(error);
    }

    if (error.response?.status === 401 && !originalRequest._retry) {
      if (originalRequest.url?.includes('/refresh-token')) {
        return Promise.reject(error);
      }

      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          requestQueue.push({ resolve, reject });
        })
          .then((token) => {
            originalRequest.headers.Authorization = `Bearer ${token}`;
            return apiClient(originalRequest);
          })
          .catch((err) => Promise.reject(err));
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        const refreshUrl = import.meta.env.VITE_API_BASE_URL 
          ? `${import.meta.env.VITE_API_BASE_URL}/refresh-token` 
          : '/api/v1/refresh-token';

        const refreshResponse = await axios.post(
          refreshUrl,
          {},
          { withCredentials: true }
        );

        const newAccessToken = refreshResponse.data.access_token;
        useAuthStore.getState().setAccessToken(newAccessToken);

        requestQueue.forEach((prom) => prom.resolve(newAccessToken));
        requestQueue = [];

        originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
        return apiClient(originalRequest);
      } catch (refreshError: any) {
        let finalError = refreshError;

        if (refreshError.response?.status === 409) {
          await new Promise((resolve) => setTimeout(resolve, 500));
          try {
            const refreshUrl = import.meta.env.VITE_API_BASE_URL 
              ? `${import.meta.env.VITE_API_BASE_URL}/refresh-token` 
              : '/api/v1/refresh-token';

            const retryRefreshResponse = await axios.post(
              refreshUrl,
              {},
              { withCredentials: true }
            );

            const newAccessToken = retryRefreshResponse.data.access_token;
            useAuthStore.getState().setAccessToken(newAccessToken);

            requestQueue.forEach((prom) => prom.resolve(newAccessToken));
            requestQueue = [];

            originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
            return apiClient(originalRequest);
          } catch (retryError: any) {
            finalError = retryError;
          }
        }

        requestQueue.forEach((prom) => prom.reject(finalError));
        requestQueue = [];

        const msg = finalError.response?.data?.message || finalError.response?.data?.error || '';
        if (msg.toLowerCase().includes('token reuse')) {
          useAuthStore.getState().setSecurityAlert(true);
        }
        
        useAuthStore.getState().clearAuth();
        window.location.href = '/login';
        return Promise.reject(finalError);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(error);
  }
);
