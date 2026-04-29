import { useNavigate } from 'react-router-dom';
import { LogOut } from 'lucide-react';
import { Button, type ButtonProps } from '@/components/ui/button';
import { authApi } from '@/api/auth';
import { useAuthStore } from '@/stores/authStore';

export function LogoutButton({ className, variant = 'outline', ...props }: ButtonProps) {
  const navigate = useNavigate();
  const clearAuth = useAuthStore((state) => state.clearAuth);

  const handleLogout = async () => {
    // Optimistically clear the auth state and navigate
    clearAuth();
    navigate('/login');

    // Fire and forget the server-side logout to invalidate the refresh token
    try {
      await authApi.logout();
    } catch (error) {
      // Ignored: the user is already logged out locally.
    }
  };

  return (
    <Button 
      variant={variant} 
      className={className} 
      onClick={handleLogout}
      {...props}
    >
      <LogOut className="mr-2 h-4 w-4" />
      Log out
    </Button>
  );
}
