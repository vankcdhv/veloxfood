'use client';

import { RoleGuard } from './role-guard';
import { ProfileForm } from './profile-form';
import { ChangePasswordForm } from './change-password-form';

export function ProfileView() {
  return (
    <RoleGuard allow={(a) => a.isAuthenticated}>
      <main className="bg-background min-h-dvh px-4 py-10">
        <div className="mx-auto w-full max-w-2xl space-y-6">
          <h1 className="font-serif text-2xl font-bold">Hồ sơ của tôi</h1>
          <ProfileForm />
          <ChangePasswordForm />
        </div>
      </main>
    </RoleGuard>
  );
}
