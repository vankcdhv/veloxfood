'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { PasswordInput } from '@/shared/ui/password-input';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { ROUTES } from '@/shared/config/constants';
import { AuthFormField } from './auth-form-field';
import { OtpVerifyForm } from './otp-verify-form';
import { registerSchema, type RegisterInput } from '../schemas/auth-schema';
import { useRegister } from '../hooks/use-auth-mutations';

export function RegisterForm() {
  const router = useRouter();
  const registerMutation = useRegister();
  // After step 1 succeeds we hold the destination (email) for OTP verification.
  const [pendingDestination, setPendingDestination] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<RegisterInput>({
    resolver: zodResolver(registerSchema),
    mode: 'onBlur',
    defaultValues: {
      full_name: '',
      email: '',
      phone: '',
      password: '',
      confirm_password: '',
      accept_terms: false,
    },
  });

  const onSubmit = async (values: RegisterInput) => {
    try {
      await registerMutation.mutateAsync({
        full_name: values.full_name,
        email: values.email,
        phone: values.phone || undefined,
        password: values.password,
      });
      setPendingDestination(values.email);
      toast.success('Đã gửi mã OTP tới email của bạn.');
    } catch (err) {
      toast.error(getApiErrorMessage(err, 'Đăng ký thất bại'));
    }
  };

  if (pendingDestination) {
    return (
      <OtpVerifyForm
        destination={pendingDestination}
        onVerified={() => router.push(ROUTES.shop.root)}
      />
    );
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
      <AuthFormField
        id="register-fullname"
        label="Họ và tên"
        required
        error={errors.full_name?.message}
      >
        <Input
          id="register-fullname"
          autoComplete="name"
          placeholder="Nguyễn Văn A"
          aria-invalid={!!errors.full_name}
          {...register('full_name')}
        />
      </AuthFormField>

      <AuthFormField id="register-email" label="Email" required error={errors.email?.message}>
        <Input
          id="register-email"
          type="email"
          autoComplete="email"
          inputMode="email"
          placeholder="ban@veloxfood.vn"
          aria-invalid={!!errors.email}
          {...register('email')}
        />
      </AuthFormField>

      <AuthFormField
        id="register-phone"
        label="Số điện thoại"
        hint="Không bắt buộc"
        error={errors.phone?.message}
      >
        <Input
          id="register-phone"
          type="tel"
          autoComplete="tel"
          inputMode="tel"
          placeholder="09xx xxx xxx"
          aria-invalid={!!errors.phone}
          {...register('phone')}
        />
      </AuthFormField>

      <AuthFormField
        id="register-password"
        label="Mật khẩu"
        required
        hint="Tối thiểu 8 ký tự, có chữ hoa và số"
        error={errors.password?.message}
      >
        <PasswordInput
          id="register-password"
          autoComplete="new-password"
          placeholder="••••••••"
          aria-invalid={!!errors.password}
          {...register('password')}
        />
      </AuthFormField>

      <AuthFormField
        id="register-confirm"
        label="Xác nhận mật khẩu"
        required
        error={errors.confirm_password?.message}
      >
        <PasswordInput
          id="register-confirm"
          autoComplete="new-password"
          placeholder="••••••••"
          aria-invalid={!!errors.confirm_password}
          {...register('confirm_password')}
        />
      </AuthFormField>

      <div className="space-y-1.5">
        <label className="flex cursor-pointer items-start gap-2 text-sm">
          <input
            type="checkbox"
            className="border-input text-primary focus-visible:ring-ring mt-0.5 h-4 w-4 cursor-pointer rounded border focus-visible:ring-1"
            aria-invalid={!!errors.accept_terms}
            {...register('accept_terms')}
          />
          <span className="text-muted-foreground">
            Tôi đồng ý với{' '}
            <a href="#" className="text-primary hover:underline">
              Điều khoản sử dụng
            </a>{' '}
            và{' '}
            <a href="#" className="text-primary hover:underline">
              Chính sách bảo mật
            </a>
            .
          </span>
        </label>
        {errors.accept_terms && (
          <p role="alert" className="text-destructive text-xs">
            {errors.accept_terms.message}
          </p>
        )}
      </div>

      <Button type="submit" className="w-full" disabled={isSubmitting}>
        {isSubmitting ? (
          <>
            <Loader2 className="h-4 w-4 animate-spin" />
            <span>Đang tạo tài khoản…</span>
          </>
        ) : (
          'Tạo tài khoản'
        )}
      </Button>
    </form>
  );
}
