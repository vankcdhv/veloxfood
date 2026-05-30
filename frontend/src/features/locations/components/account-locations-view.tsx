'use client';

import { RoleGuard } from '@/features/auth/components/role-guard';
import { MyLocationsView } from './my-locations-view';

// Client wrapper keeping the RoleGuard predicate inside the client boundary.
export function AccountLocationsView() {
  return (
    <RoleGuard allow={(a) => a.isAuthenticated}>
      <main className="bg-background min-h-dvh px-4 py-10">
        <div className="mx-auto w-full max-w-2xl">
          <h1 className="font-serif mb-6 text-2xl font-bold">Vị trí giao của tôi</h1>
          <MyLocationsView />
        </div>
      </main>
    </RoleGuard>
  );
}
