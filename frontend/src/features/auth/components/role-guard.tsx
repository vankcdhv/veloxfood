'use client';

import { useEffect, type ReactNode } from 'react';
import { useRouter } from 'next/navigation';
import { Loader2 } from 'lucide-react';
import { ROUTES } from '@/shared/config/constants';
import { useAuth } from '../context/auth-provider';
import type { SessionState } from '../hooks/use-session';

interface RoleGuardProps {
  children: ReactNode;
  // allow returns true when the current session may view the guarded subtree.
  allow: (auth: SessionState) => boolean;
  // where to send authenticated-but-unauthorized users (default: storefront).
  fallback?: string;
  // where to send unauthenticated users (default: login, preserving ?next).
  loginNext?: string;
}

// RoleGuard enforces role/permission access client-side. The Next middleware
// only checks cookie presence; role verification needs the /me payload which is
// only available in the browser.
export function RoleGuard({ children, allow, fallback = ROUTES.shop.root, loginNext }: RoleGuardProps) {
  const router = useRouter();
  const auth = useAuth();
  const { isLoading, isAuthenticated } = auth;
  const allowed = allow(auth);

  useEffect(() => {
    if (isLoading) return;
    if (!isAuthenticated) {
      const next = loginNext ? `?next=${loginNext}` : '';
      router.replace(`${ROUTES.auth.login}${next}`);
    } else if (!allowed) {
      router.replace(fallback);
    }
  }, [isLoading, isAuthenticated, allowed, fallback, loginNext, router]);

  if (isLoading || !isAuthenticated || !allowed) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="text-muted-foreground h-6 w-6 animate-spin" />
      </div>
    );
  }

  return <>{children}</>;
}
