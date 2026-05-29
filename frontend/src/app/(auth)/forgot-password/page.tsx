import type { Metadata } from 'next';
import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';
import { ForgotPasswordForm } from '@/features/auth/components/forgot-password-form';
import { ROUTES } from '@/shared/config/constants';

export const metadata: Metadata = {
  title: 'Quên mật khẩu',
  description: 'Yêu cầu link đặt lại mật khẩu cho tài khoản VeloxFood của bạn.',
};

export default function ForgotPasswordPage() {
  return (
    <div className="space-y-6">
      <header className="space-y-2 text-center sm:text-left">
        <h1 className="font-serif text-3xl font-semibold tracking-tight">Quên mật khẩu?</h1>
        <p className="text-muted-foreground text-sm">
          Đừng lo — nhập email của bạn, chúng tôi sẽ gửi link đặt lại mật khẩu trong vài phút.
        </p>
      </header>

      <ForgotPasswordForm />

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
