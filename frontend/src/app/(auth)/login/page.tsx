import type { Metadata } from 'next';
import { Suspense } from 'react';
import Link from 'next/link';
import { LoginForm } from '@/features/auth/components/login-form';
import { ROUTES } from '@/shared/config/constants';

export const metadata: Metadata = {
  title: 'Đăng nhập',
  description: 'Đăng nhập vào VeloxFood để đặt món và theo dõi đơn hàng.',
};

export default function LoginPage() {
  return (
    <div className="space-y-6">
      <header className="space-y-2 text-center sm:text-left">
        <h1 className="font-serif text-3xl font-semibold tracking-tight">Đăng nhập</h1>
        <p className="text-muted-foreground text-sm">
          Chào mừng trở lại! Đăng nhập để tiếp tục đặt món yêu thích.
        </p>
      </header>

      <Suspense fallback={<div className="text-muted-foreground text-sm">Đang tải…</div>}>
        <LoginForm />
      </Suspense>

      <p className="text-muted-foreground text-center text-sm">
        Chưa có tài khoản?{' '}
        <Link
          href={ROUTES.auth.register}
          className="text-primary hover:text-primary/80 font-medium underline-offset-4 hover:underline"
        >
          Đăng ký ngay
        </Link>
      </p>
    </div>
  );
}
