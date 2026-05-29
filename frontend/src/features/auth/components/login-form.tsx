'use client';

import { useRouter, useSearchParams } from 'next/navigation';
import { useQueryClient } from '@tanstack/react-query';
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
import { loginSchema, type LoginInput } from '../schemas/auth-schema';
import { useLogin } from '../hooks/use-auth-mutations';
import { getMe } from '../api/auth-api';
import { ME_QUERY_KEY } from '../hooks/use-session';

const ADMIN_ROLE_CODES = new Set(['SUPER_ADMIN', 'SCHOOL_ADMIN']);

export function LoginForm() {
  const router = useRouter();
  const params = useSearchParams();
  const queryClient = useQueryClient();
  const loginMutation = useLogin();

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginInput>({
    resolver: zodResolver(loginSchema),
    mode: 'onBlur',
    defaultValues: { email: '', password: '', remember: false },
  });

  const onSubmit = async (values: LoginInput) => {
    try {
      await loginMutation.mutateAsync({ identifier: values.email, password: values.password });

      const next = params.get('next');
      if (next) {
        router.push(next);
        return;
      }
      // No explicit target — route admins to the dashboard, everyone else to shop.
      const me = await queryClient.fetchQuery({ queryKey: ME_QUERY_KEY, queryFn: getMe });
      const isAdmin = me.roles.some(
        (r) => r.scope_type === 'global' && ADMIN_ROLE_CODES.has(r.role_code),
      );
      router.push(isAdmin ? ROUTES.admin.root : ROUTES.shop.root);
    } catch (err) {
      toast.error(getApiErrorMessage(err, 'Đăng nhập thất bại'));
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
      <AuthFormField id="login-email" label="Email" required error={errors.email?.message}>
        <Input
          id="login-email"
          type="email"
          autoComplete="email"
          inputMode="email"
          placeholder="ban@veloxfood.vn"
          aria-invalid={!!errors.email}
          aria-describedby={errors.email ? 'login-email-error' : undefined}
          {...register('email')}
        />
      </AuthFormField>

      <AuthFormField id="login-password" label="Mật khẩu" required error={errors.password?.message}>
        <PasswordInput
          id="login-password"
          autoComplete="current-password"
          placeholder="••••••••"
          aria-invalid={!!errors.password}
          aria-describedby={errors.password ? 'login-password-error' : undefined}
          {...register('password')}
        />
      </AuthFormField>

      <div className="flex items-center justify-between text-sm">
        <label className="flex cursor-pointer items-center gap-2">
          <input
            type="checkbox"
            className="border-input text-primary focus-visible:ring-ring h-4 w-4 cursor-pointer rounded border focus-visible:ring-1"
            {...register('remember')}
          />
          <span className="text-muted-foreground">Ghi nhớ đăng nhập</span>
        </label>
        <a
          href={ROUTES.auth.forgotPassword}
          className="text-primary hover:text-primary/80 font-medium transition-colors"
        >
          Quên mật khẩu?
        </a>
      </div>

      <Button type="submit" className="w-full" disabled={isSubmitting}>
        {isSubmitting ? (
          <>
            <Loader2 className="h-4 w-4 animate-spin" />
            <span>Đang xử lý…</span>
          </>
        ) : (
          'Đăng nhập'
        )}
      </Button>
    </form>
  );
}
