import { LogOut } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useAuthStore } from '@/stores/authStore';
import { authApi } from '@/api/auth';
import { useNavigate } from 'react-router-dom';

interface LogoutButtonProps {
  variant?: "default" | "destructive" | "outline" | "secondary" | "ghost" | "link";
  className?: string;
  showIcon?: boolean;
  showText?: boolean;
}

export function LogoutButton({ variant = "ghost", className, showIcon = true, showText = true }: LogoutButtonProps) {
  const { clearAuth } = useAuthStore();
  const navigate = useNavigate();

  const handleLogout = async () => {
    // Fire and forget
    authApi.logout().catch(() => {});
    clearAuth();
    navigate('/login', { replace: true });
  };

  return (
    <Button variant={variant} className={className} onClick={handleLogout}>
      {showIcon && <LogOut className={`h-4 w-4 ${showText ? "mr-2" : ""}`} />}
      {showText && "Logout"}
    </Button>
  );
}
