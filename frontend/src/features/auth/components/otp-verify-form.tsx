'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { AuthFormField } from './auth-form-field';
import { verifyOtpSchema, type VerifyOtpInput } from '../schemas/auth-schema';
import { useVerifyRegister } from '../hooks/use-auth-mutations';

interface OtpVerifyFormProps {
  destination: string;
  onVerified: () => void;
}

export function OtpVerifyForm({ destination, onVerified }: OtpVerifyFormProps) {
  const verifyMutation = useVerifyRegister();

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<VerifyOtpInput>({
    resolver: zodResolver(verifyOtpSchema),
    mode: 'onBlur',
    defaultValues: { code: '' },
  });

  const onSubmit = async (values: VerifyOtpInput) => {
    try {
      await verifyMutation.mutateAsync({ destination, code: values.code });
      toast.success('Xác thực thành công! Đang đăng nhập…');
      onVerified();
    } catch (err) {
      toast.error(getApiErrorMessage(err, 'Mã OTP không đúng hoặc đã hết hạn'));
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
      <p className="text-muted-foreground text-sm">
        Chúng tôi đã gửi mã xác thực 6 chữ số tới <span className="font-medium">{destination}</span>.
        Nhập mã để hoàn tất đăng ký.
      </p>

      <AuthFormField id="otp-code" label="Mã OTP" required error={errors.code?.message}>
        <Input
          id="otp-code"
          inputMode="numeric"
          autoComplete="one-time-code"
          maxLength={6}
          placeholder="123456"
          aria-invalid={!!errors.code}
          {...register('code')}
        />
      </AuthFormField>

      <Button type="submit" className="w-full" disabled={isSubmitting}>
        {isSubmitting ? (
          <>
            <Loader2 className="h-4 w-4 animate-spin" />
            <span>Đang xác thực…</span>
          </>
        ) : (
          'Xác nhận'
        )}
      </Button>
    </form>
  );
}
