import { useEffect, useRef } from 'react';
import axios from 'axios';
import { useAuthStore } from '@/stores/authStore';
import { apiClient } from '@/api/client';

// Module-level flag prevents StrictMode double-invocation from firing two
// simultaneous /refresh-token requests (which triggers backend token-reuse detection).
let restoreInProgress = false;

const refreshUrl = import.meta.env.VITE_API_BASE_URL
  ? `${import.meta.env.VITE_API_BASE_URL}/refresh-token`
  : '/api/v1/refresh-token';

export function useSilentRestore() {
  const { setAccessToken, setUser, setStatus } = useAuthStore();
  const isMounted = useRef(true);

  useEffect(() => {
    isMounted.current = true;

    if (useAuthStore.getState().status === 'authenticated') return;
    if (restoreInProgress) return;

    restoreInProgress = true;

    const restore = async () => {
      try {
        // Use raw axios (not apiClient) to bypass the response interceptor entirely.
        // The interceptor is designed for authenticated requests — it must not interfere
        // with the unauthenticated restore flow.
        let refreshResponse;
        try {
          refreshResponse = await axios.post(refreshUrl, {}, { withCredentials: true });
        } catch (err: any) {
          // 409 = concurrent refresh (e.g. two tabs refreshed simultaneously).
          // Retry once after a short delay — the other tab's rotation will have
          // written the new cookie by then.
          if (err.response?.status === 409) {
            await new Promise((r) => setTimeout(r, 600));
            refreshResponse = await axios.post(refreshUrl, {}, { withCredentials: true });
          } else {
            throw err;
          }
        }

        const token: string = refreshResponse.data.access_token;

        if (!isMounted.current) return;
        setAccessToken(token);

        // Fetch user profile with the new access token via the normal client.
        const userResponse = await apiClient.get('/user', {
          headers: { Authorization: `Bearer ${token}` },
        });

        if (!isMounted.current) return;
        setUser(userResponse.data);
        setStatus('authenticated');
      } catch {
        if (isMounted.current) {
          setStatus('unauthenticated');
        }
      } finally {
        restoreInProgress = false;
      }
    };

    restore();

    return () => {
      isMounted.current = false;
    };
  }, [setAccessToken, setUser, setStatus]);
}
