'use client';

import { useRouter } from 'next/navigation';
import { LogOut, LayoutDashboard, User as UserIcon, Wallet } from 'lucide-react';
import { toast } from 'sonner';
import { Avatar, AvatarFallback } from '@/shared/ui/avatar';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from '@/shared/ui/dropdown-menu';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { ROUTES } from '@/shared/config/constants';
import { useAuth } from '../context/auth-provider';
import { useLogout } from '../hooks/use-auth-mutations';

function initialsOf(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return 'U';
  const first = parts[0]![0] ?? '';
  const last = parts.length > 1 ? (parts[parts.length - 1]![0] ?? '') : '';
  return (first + last).toUpperCase() || 'U';
}

interface UserMenuProps {
  // Show a link into the admin dashboard (only rendered for admins).
  showAdminLink?: boolean;
  align?: 'end' | 'start' | 'center';
}

// UserMenu renders the avatar dropdown for an authenticated user (name, email,
// optional admin entry, logout). Shared by the shop header and admin topbar.
export function UserMenu({ showAdminLink = false, align = 'end' }: UserMenuProps) {
  const router = useRouter();
  const { user, isAdmin } = useAuth();
  const logoutMutation = useLogout();

  const handleLogout = async () => {
    try {
      await logoutMutation.mutateAsync();
    } catch (err) {
      toast.error(getApiErrorMessage(err, 'Đăng xuất thất bại'));
    } finally {
      router.push(ROUTES.auth.login);
    }
  };

  const displayName = user?.full_name || 'Tài khoản';

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          aria-label="Tài khoản"
          className="focus-visible:ring-ring rounded-full focus-visible:ring-2 focus-visible:outline-none"
        >
          <Avatar>
            <AvatarFallback>{initialsOf(displayName)}</AvatarFallback>
          </Avatar>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align={align} className="w-56">
        <DropdownMenuLabel>
          <div className="flex flex-col">
            <span className="truncate text-sm font-medium">{displayName}</span>
            {user?.email && (
              <span className="text-muted-foreground truncate text-xs font-normal">
                {user.email}
              </span>
            )}
          </div>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => router.push(ROUTES.shop.root)} className="cursor-pointer">
          <UserIcon className="mr-2 h-4 w-4" />
          Cửa hàng
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => router.push(ROUTES.account.wallet)} className="cursor-pointer">
          <Wallet className="mr-2 h-4 w-4" />
          Ví của tôi
        </DropdownMenuItem>
        {showAdminLink && isAdmin && (
          <DropdownMenuItem
            onClick={() => router.push(ROUTES.admin.root)}
            className="cursor-pointer"
          >
            <LayoutDashboard className="mr-2 h-4 w-4" />
            Trang quản trị
          </DropdownMenuItem>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuItem
          onClick={handleLogout}
          disabled={logoutMutation.isPending}
          className="text-destructive focus:text-destructive cursor-pointer"
        >
          <LogOut className="mr-2 h-4 w-4" />
          Đăng xuất
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
