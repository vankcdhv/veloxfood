'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { AuthFormField } from './auth-form-field';
import { forgotPasswordSchema, type ForgotPasswordInput } from '../schemas/auth-schema';
import { useForgotPassword } from '../hooks/use-auth-mutations';

export function ForgotPasswordForm() {
  const forgotMutation = useForgotPassword();

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ForgotPasswordInput>({
    resolver: zodResolver(forgotPasswordSchema),
    mode: 'onBlur',
    defaultValues: { email: '' },
  });

  const onSubmit = async (values: ForgotPasswordInput) => {
    // Backend always responds success (no user enumeration) — mirror that here.
    await forgotMutation.mutateAsync({ identifier: values.email }).catch(() => {});
    toast.success('Nếu tài khoản tồn tại, chúng tôi đã gửi link đặt lại mật khẩu vào email.');
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
      <AuthFormField
        id="forgot-email"
        label="Email"
        required
        hint="Chúng tôi sẽ gửi link đặt lại mật khẩu vào email này."
        error={errors.email?.message}
      >
        <Input
          id="forgot-email"
          type="email"
          autoComplete="email"
          inputMode="email"
          placeholder="ban@veloxfood.vn"
          aria-invalid={!!errors.email}
          {...register('email')}
        />
      </AuthFormField>

      <Button type="submit" className="w-full" disabled={isSubmitting}>
        {isSubmitting ? (
          <>
            <Loader2 className="h-4 w-4 animate-spin" />
            <span>Đang gửi…</span>
          </>
        ) : (
          'Gửi link đặt lại'
        )}
      </Button>
    </form>
  );
}
