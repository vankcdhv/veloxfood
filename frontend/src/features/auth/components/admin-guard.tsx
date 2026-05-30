'use client';

import { type ReactNode } from 'react';
import { ROUTES } from '@/shared/config/constants';
import { RoleGuard } from './role-guard';

// AdminGuard is a thin wrapper over RoleGuard for the /admin area.
export function AdminGuard({ children }: { children: ReactNode }) {
  return (
    <RoleGuard allow={(a) => a.isAdmin} fallback={ROUTES.shop.root} loginNext={ROUTES.admin.root}>
      {children}
    </RoleGuard>
  );
}
