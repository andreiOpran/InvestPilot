import { useEffect } from 'react';
import { useAuthStore } from '@/stores/authStore';
import { authApi } from '@/api/auth';
import { userApi } from '@/api/user';

export function useSilentRestore() {
  const { setAccessToken, setUser, setStatus } = useAuthStore();

  useEffect(() => {
    // Read the LIVE store state at effect execution time (not stale closure).
    // If we're already authenticated (e.g. right after a fresh login), skip
    // the restore flow entirely to prevent a race that would blank protected pages.
    const currentStatus = useAuthStore.getState().status;
    if (currentStatus === 'authenticated') return;

    let isMounted = true;

    const restoreSession = async () => {
      try {
        const refreshResponse = await authApi.refreshToken();
        const token = refreshResponse.data.access_token;

        if (isMounted) {
          setAccessToken(token);
        }

        const userResponse = await userApi.getUser();

        if (isMounted) {
          setUser(userResponse.data);
          setStatus('authenticated');
        }
      } catch {
        if (isMounted) {
          setStatus('unauthenticated');
        }
      }
    };

    restoreSession();

    return () => {
      isMounted = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // Run only on mount — live store state is read inside the effect
}


