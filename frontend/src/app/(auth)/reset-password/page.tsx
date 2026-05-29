import type { Metadata } from 'next';
import { Suspense } from 'react';
import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';
import { ResetPasswordForm } from '@/features/auth/components/reset-password-form';
import { ROUTES } from '@/shared/config/constants';

export const metadata: Metadata = {
  title: 'Đặt lại mật khẩu',
  description: 'Đặt mật khẩu mới cho tài khoản VeloxFood.',
};

export default function ResetPasswordPage() {
  return (
    <div className="space-y-6">
      <header className="space-y-2 text-center sm:text-left">
        <h1 className="font-serif text-3xl font-semibold tracking-tight">Đặt lại mật khẩu</h1>
        <p className="text-muted-foreground text-sm">
          Chọn mật khẩu mới mạnh và dễ nhớ. Hãy giữ kín thông tin này.
        </p>
      </header>

      <Suspense fallback={<div className="text-muted-foreground text-sm">Đang tải…</div>}>
        <ResetPasswordForm />
      </Suspense>

      <Link
        href={ROUTES.auth.login}
        className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1.5 text-sm transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        Quay lại đăng nhập
      </Link>
    </div>
  );
}
