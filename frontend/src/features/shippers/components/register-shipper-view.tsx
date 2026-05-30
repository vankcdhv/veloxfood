'use client';

import { RoleGuard } from '@/features/auth/components/role-guard';
import { ShipperRegistrationForm } from './shipper-registration-form';

// Client wrapper so the RoleGuard `allow` predicate stays inside the client
// boundary (functions cannot be passed from a Server Component to a Client one).
export function RegisterShipperView() {
  return (
    <RoleGuard allow={(a) => a.isAuthenticated}>
      <main className="bg-background min-h-dvh px-4 py-10">
        <div className="mx-auto w-full max-w-lg">
          <h1 className="font-serif mb-6 text-center text-2xl font-bold">Trở thành Shipper</h1>
          <ShipperRegistrationForm />
        </div>
      </main>
    </RoleGuard>
  );
}
