'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { AuthFormField } from './auth-form-field';
import { updateProfileSchema, type UpdateProfileInput } from '../schemas/auth-schema';
import { useUpdateProfile } from '../hooks/use-auth-mutations';
import { useSession } from '../hooks/use-session';

export function ProfileForm() {
  const { user } = useSession();
  const update = useUpdateProfile();

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<UpdateProfileInput>({
    resolver: zodResolver(updateProfileSchema),
    mode: 'onBlur',
    // Keep the form synced with the session once /me resolves.
    values: { full_name: user?.full_name ?? '' },
  });

  const onSubmit = async (v: UpdateProfileInput) => {
    try {
      await update.mutateAsync({ full_name: v.full_name.trim() });
      toast.success('Đã cập nhật hồ sơ.');
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Cập nhật hồ sơ thất bại'));
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Thông tin cá nhân</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
          <AuthFormField id="profile-name" label="Họ và tên" required error={errors.full_name?.message}>
            <Input id="profile-name" placeholder="Nguyễn Văn A" {...register('full_name')} />
          </AuthFormField>

          {/* Email & phone are managed elsewhere — shown read-only for reference. */}
          <div className="grid gap-1.5">
            <label className="text-foreground text-sm font-medium">Email</label>
            <Input value={user?.email ?? '—'} readOnly disabled />
          </div>
          <div className="grid gap-1.5">
            <label className="text-foreground text-sm font-medium">Số điện thoại</label>
            <Input value={user?.phone || '—'} readOnly disabled />
          </div>

          <Button type="submit" disabled={isSubmitting || update.isPending}>
            {isSubmitting || update.isPending ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                <span>Đang lưu…</span>
              </>
            ) : (
              'Lưu thay đổi'
            )}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
