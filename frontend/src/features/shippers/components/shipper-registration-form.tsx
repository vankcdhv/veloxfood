'use client';

import { useState } from 'react';
import { CheckCircle2, Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { PhotoUpload } from '@/shared/ui/photo-upload';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/card';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useRegisterShipper } from '../hooks/use-shippers';

export function ShipperRegistrationForm() {
  const [idDoc, setIdDoc] = useState<File | null>(null);
  const [portrait, setPortrait] = useState<File | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [done, setDone] = useState(false);
  const mutation = useRegisterShipper();

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitError(null);
    if (!idDoc || !portrait) {
      setSubmitError('Vui lòng tải lên cả ảnh giấy tờ và ảnh chân dung.');
      return;
    }
    try {
      await mutation.mutateAsync({ idDocument: idDoc, portrait });
      setDone(true);
      toast.success('Đã gửi hồ sơ shipper — chờ quản trị viên duyệt.');
    } catch (err) {
      setSubmitError(getApiErrorMessage(err, 'Gửi hồ sơ thất bại'));
    }
  };

  if (done) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center gap-3 py-10 text-center">
          <CheckCircle2 className="text-primary h-12 w-12" aria-hidden="true" />
          <h2 className="text-xl font-semibold">Hồ sơ đã được gửi</h2>
          <p className="text-muted-foreground max-w-sm text-sm">
            Quản trị viên sẽ xem xét giấy tờ của bạn và phản hồi sớm. Bạn sẽ nhận được thông báo khi
            hồ sơ được duyệt.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Đăng ký làm Shipper</CardTitle>
        <CardDescription>
          Tải lên ảnh giấy tờ tuỳ thân và ảnh chân dung để quản trị viên xác minh.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={onSubmit} className="space-y-5" noValidate>
          <PhotoUpload label="Ảnh giấy tờ tuỳ thân" value={idDoc} onChange={setIdDoc} required />
          <PhotoUpload label="Ảnh chân dung" value={portrait} onChange={setPortrait} required />

          {submitError && (
            <p role="alert" className="text-destructive text-sm">
              {submitError}
            </p>
          )}

          <Button type="submit" className="w-full" disabled={mutation.isPending}>
            {mutation.isPending ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                <span>Đang gửi…</span>
              </>
            ) : (
              'Gửi hồ sơ'
            )}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
