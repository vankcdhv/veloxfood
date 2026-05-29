'use client';

import { useEffect, type ReactNode } from 'react';
import { useRouter } from 'next/navigation';
import { Loader2 } from 'lucide-react';
import { ROUTES } from '@/shared/config/constants';
import { useAuth } from '../context/auth-provider';

// AdminGuard enforces the admin role client-side. The Next middleware only
// checks cookie presence; role verification needs the /me payload which is
// only available in the browser.
export function AdminGuard({ children }: { children: ReactNode }) {
  const router = useRouter();
  const { isLoading, isAuthenticated, isAdmin } = useAuth();

  useEffect(() => {
    if (isLoading) return;
    if (!isAuthenticated) {
      router.replace(`${ROUTES.auth.login}?next=${ROUTES.admin.root}`);
    } else if (!isAdmin) {
      router.replace(ROUTES.shop.root);
    }
  }, [isLoading, isAuthenticated, isAdmin, router]);

  if (isLoading || !isAuthenticated || !isAdmin) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="text-muted-foreground h-6 w-6 animate-spin" />
      </div>
    );
  }

  return <>{children}</>;
}
