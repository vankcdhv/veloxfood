'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { PasswordInput } from '@/shared/ui/password-input';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { AuthFormField } from './auth-form-field';
import { changePasswordSchema, type ChangePasswordInput } from '../schemas/auth-schema';
import { useChangePassword } from '../hooks/use-auth-mutations';
import { useSession } from '../hooks/use-session';

export function ChangePasswordForm() {
  const { user } = useSession();
  const change = useChangePassword();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<ChangePasswordInput>({
    resolver: zodResolver(changePasswordSchema),
    mode: 'onBlur',
    defaultValues: { old_password: '', password: '', confirm_password: '' },
  });

  const onSubmit = async (v: ChangePasswordInput) => {
    if (!user?.id) return;
    try {
      await change.mutateAsync({
        user_id: user.id,
        old_password: v.old_password,
        new_password: v.password,
      });
      toast.success('Đã đổi mật khẩu.');
      reset();
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Đổi mật khẩu thất bại'));
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Đổi mật khẩu</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
          <AuthFormField id="cp-old" label="Mật khẩu hiện tại" required error={errors.old_password?.message}>
            <PasswordInput id="cp-old" autoComplete="current-password" placeholder="••••••••" {...register('old_password')} />
          </AuthFormField>
          <AuthFormField id="cp-new" label="Mật khẩu mới" required hint="Tối thiểu 8 ký tự, có chữ hoa và số" error={errors.password?.message}>
            <PasswordInput id="cp-new" autoComplete="new-password" placeholder="••••••••" {...register('password')} />
          </AuthFormField>
          <AuthFormField id="cp-confirm" label="Xác nhận mật khẩu mới" required error={errors.confirm_password?.message}>
            <PasswordInput id="cp-confirm" autoComplete="new-password" placeholder="••••••••" {...register('confirm_password')} />
          </AuthFormField>

          <Button type="submit" disabled={isSubmitting || change.isPending}>
            {isSubmitting || change.isPending ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                <span>Đang đổi…</span>
              </>
            ) : (
              'Đổi mật khẩu'
            )}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
