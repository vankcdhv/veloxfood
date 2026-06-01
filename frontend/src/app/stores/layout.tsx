'use client';

import type { ReactNode } from 'react';
import { CustomerShell } from '@/widgets/customer-shell/customer-shell';
import { RoleGuard } from '@/features/auth/components/role-guard';

// Browsing stores requires a session: the backend store endpoints sit behind
// auth, so an unauthenticated visitor would otherwise hit a 401 error screen.
// Mirror the cart's behaviour and bounce them to login instead.
export default function StoresLayout({ children }: { children: ReactNode }) {
  return (
    <CustomerShell>
      <RoleGuard allow={(a) => a.isAuthenticated}>{children}</RoleGuard>
    </CustomerShell>
  );
}
