'use client';

import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Loader2, AlertCircle } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { PasswordInput } from '@/shared/ui/password-input';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { AuthFormField } from './auth-form-field';
import { resetPasswordSchema, type ResetPasswordInput } from '../schemas/auth-schema';
import { useResetPassword } from '../hooks/use-auth-mutations';
import { ROUTES } from '@/shared/config/constants';

export function ResetPasswordForm() {
  const router = useRouter();
  const params = useSearchParams();
  const token = params.get('token');
  const resetMutation = useResetPassword();

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ResetPasswordInput>({
    resolver: zodResolver(resetPasswordSchema),
    mode: 'onBlur',
    defaultValues: { password: '', confirm_password: '' },
  });

  const onSubmit = async (values: ResetPasswordInput) => {
    if (!token) return;
    try {
      await resetMutation.mutateAsync({ token, new_password: values.password });
      toast.success('Đặt lại mật khẩu thành công. Vui lòng đăng nhập lại.');
      router.push(ROUTES.auth.login);
    } catch (err) {
      toast.error(getApiErrorMessage(err, 'Đặt lại mật khẩu thất bại'));
    }
  };

  if (!token) {
    return (
      <div
        role="alert"
        className="border-destructive/30 bg-destructive/5 text-destructive flex items-start gap-3 rounded-md border p-4 text-sm"
      >
        <AlertCircle className="mt-0.5 h-5 w-5 shrink-0" />
        <div className="space-y-2">
          <p className="font-medium">Link đặt lại không hợp lệ</p>
          <p className="text-destructive/80 text-xs">
            Token không tồn tại hoặc đã hết hạn. Vui lòng yêu cầu link mới.
          </p>
          <Link
            href={ROUTES.auth.forgotPassword}
            className="text-primary hover:text-primary/80 inline-flex text-sm font-medium underline-offset-4 hover:underline"
          >
            Yêu cầu link mới →
          </Link>
        </div>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
      <AuthFormField
        id="reset-password"
        label="Mật khẩu mới"
        required
        hint="Tối thiểu 8 ký tự, có chữ hoa và số"
        error={errors.password?.message}
      >
        <PasswordInput
          id="reset-password"
          autoComplete="new-password"
          placeholder="••••••••"
          aria-invalid={!!errors.password}
          {...register('password')}
        />
      </AuthFormField>

      <AuthFormField
        id="reset-confirm"
        label="Xác nhận mật khẩu"
        required
        error={errors.confirm_password?.message}
      >
        <PasswordInput
          id="reset-confirm"
          autoComplete="new-password"
          placeholder="••••••••"
          aria-invalid={!!errors.confirm_password}
          {...register('confirm_password')}
        />
      </AuthFormField>

      <Button type="submit" className="w-full" disabled={isSubmitting}>
        {isSubmitting ? (
          <>
            <Loader2 className="h-4 w-4 animate-spin" />
            <span>Đang cập nhật…</span>
          </>
        ) : (
          'Cập nhật mật khẩu'
        )}
      </Button>
    </form>
  );
}
