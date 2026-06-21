'use client';

import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { Store, CheckCircle2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { onboardSeller } from '../api/seller-api';

// Lets a logged-in customer apply to become a store owner. The application goes
// to admin review; once approved the admin creates the store and grants manage
// rights. Mirrors the shipper registration flow.
export function RegisterSellerView() {
  const [form, setForm] = useState({ vendor_name: '', business_type: '', address: '', phone: '' });
  const [submitted, setSubmitted] = useState(false);

  const onboard = useMutation({
    mutationFn: () => onboardSeller(form),
    onSuccess: () => setSubmitted(true),
    onError: (e) => toast.error(getApiErrorMessage(e, 'Gửi hồ sơ thất bại')),
  });

  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm((f) => ({ ...f, [k]: e.target.value }));

  const canSubmit =
    form.vendor_name.trim() && form.business_type.trim() && form.address.trim() && form.phone.trim();

  return (
    <main className="bg-background min-h-dvh px-4 py-10">
      <div className="mx-auto w-full max-w-lg">
        <h1 className="font-serif mb-6 text-center text-2xl font-bold">Đăng ký bán hàng</h1>
        <Card>
          {submitted ? (
            <CardContent className="flex flex-col items-center gap-3 py-12 text-center">
              <CheckCircle2 className="text-primary h-12 w-12" />
              <p className="font-semibold">Hồ sơ đã gửi — đang chờ duyệt</p>
              <p className="text-muted-foreground text-sm">
                Quản trị viên sẽ xem xét và phê duyệt cửa hàng của bạn. Bạn sẽ nhận được thông báo khi
                có kết quả.
              </p>
            </CardContent>
          ) : (
            <>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <Store className="text-primary h-5 w-5" /> Mở cửa hàng trên VeloxFood
                </CardTitle>
                <p className="text-muted-foreground text-sm">
                  Điền thông tin cửa hàng để gửi hồ sơ. Quản trị viên sẽ duyệt trước khi cửa hàng hoạt
                  động.
                </p>
              </CardHeader>
              <CardContent className="space-y-3">
                <Field label="Tên cửa hàng">
                  <Input value={form.vendor_name} onChange={set('vendor_name')} placeholder="VD: Cơm Tấm Cô Ba" />
                </Field>
                <Field label="Loại hình">
                  <Input value={form.business_type} onChange={set('business_type')} placeholder="VD: Cơm, Bún, Đồ uống…" />
                </Field>
                <Field label="Địa chỉ">
                  <Input value={form.address} onChange={set('address')} placeholder="VD: Toà A, Tầng 1, Quầy 5" />
                </Field>
                <Field label="Số điện thoại">
                  <Input value={form.phone} onChange={set('phone')} placeholder="VD: 0901234567" />
                </Field>
                <Button
                  className="w-full"
                  disabled={!canSubmit || onboard.isPending}
                  onClick={() => onboard.mutate()}
                >
                  {onboard.isPending ? 'Đang gửi…' : 'Gửi hồ sơ'}
                </Button>
              </CardContent>
            </>
          )}
        </Card>
      </div>
    </main>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="text-foreground mb-1.5 block text-sm font-medium">{label}</label>
      {children}
    </div>
  );
}
