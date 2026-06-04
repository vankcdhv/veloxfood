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
import { getMe, loginWithGoogle } from '../api/auth-api';
import { ME_QUERY_KEY } from '../hooks/use-session';

const ADMIN_ROLE_CODES = new Set(['SUPER_ADMIN', 'ADMIN']);

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
      <AuthFormField id="login-email" label="Email hoặc số điện thoại" required error={errors.email?.message}>
        <Input
          id="login-email"
          type="text"
          autoComplete="username"
          placeholder="ban@veloxfood.vn hoặc 09xxxxxxxx"
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

      <div className="flex items-center gap-3" aria-hidden="true">
        <span className="bg-border h-px flex-1" />
        <span className="text-muted-foreground text-xs">hoặc</span>
        <span className="bg-border h-px flex-1" />
      </div>

      <Button
        type="button"
        variant="outline"
        className="w-full"
        onClick={() => loginWithGoogle()}
      >
        <GoogleMark className="h-4 w-4" />
        <span>Đăng nhập với Google</span>
      </Button>
    </form>
  );
}

// GoogleMark renders the multicolour Google "G" (no emoji — see design-system rules).
function GoogleMark({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" aria-hidden="true">
      <path
        fill="#4285F4"
        d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.76h3.57c2.08-1.92 3.27-4.74 3.27-8.09Z"
      />
      <path
        fill="#34A853"
        d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.76c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84A11 11 0 0 0 12 23Z"
      />
      <path
        fill="#FBBC05"
        d="M5.84 14.1a6.6 6.6 0 0 1 0-4.2V7.06H2.18a11 11 0 0 0 0 9.88l3.66-2.84Z"
      />
      <path
        fill="#EA4335"
        d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.06l3.66 2.84C6.71 7.3 9.14 5.38 12 5.38Z"
      />
    </svg>
  );
}
