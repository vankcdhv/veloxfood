import type { Metadata } from 'next';
import Link from 'next/link';
import { RegisterForm } from '@/features/auth/components/register-form';
import { ROUTES } from '@/shared/config/constants';

export const metadata: Metadata = {
  title: 'Đăng ký',
  description: 'Tạo tài khoản VeloxFood để đặt món và nhận ưu đãi thành viên.',
};

export default function RegisterPage() {
  return (
    <div className="space-y-6">
      <header className="space-y-2 text-center sm:text-left">
        <h1 className="font-serif text-3xl font-semibold tracking-tight">Tạo tài khoản</h1>
        <p className="text-muted-foreground text-sm">
          Vài thông tin nhỏ là bạn đã có thể đặt món và nhận ưu đãi.
        </p>
      </header>

      <RegisterForm />

      <p className="text-muted-foreground text-center text-sm">
        Đã có tài khoản?{' '}
        <Link
          href={ROUTES.auth.login}
          className="text-primary hover:text-primary/80 font-medium underline-offset-4 hover:underline"
        >
          Đăng nhập
        </Link>
      </p>
    </div>
  );
}
