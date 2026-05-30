'use client';

import { useState } from 'react';
import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useVendorStoreMutations } from '../hooks/use-stores';

interface Props {
  storeId: string;
}

// Minimal hours-change request form. The payload is freeform per the backend
// contract (POST /stores/:id/hours-change { payload: {...} }). We capture
// open/close times per weekday and forward as-is.
export function VendorHoursChangeForm({ storeId }: Props) {
  const m = useVendorStoreMutations(storeId);
  const [openTime, setOpenTime] = useState('');
  const [closeTime, setCloseTime] = useState('');
  const [note, setNote] = useState('');

  const submit = () => {
    if (!openTime || !closeTime) {
      toast.error('Vui lòng nhập giờ mở và đóng cửa.');
      return;
    }
    const payload: Record<string, unknown> = {
      open_time: openTime,
      close_time: closeTime,
      note: note.trim(),
    };
    m.requestHoursChange.mutate(payload, {
      onSuccess: () => {
        toast.success('Đã gửi yêu cầu thay đổi giờ. Chờ admin duyệt.');
        setOpenTime('');
        setCloseTime('');
        setNote('');
      },
      onError: (e) => toast.error(getApiErrorMessage(e, 'Gửi yêu cầu thất bại')),
    });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Yêu cầu thay đổi giờ hoạt động</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="text-foreground mb-1.5 block text-sm font-medium">Giờ mở cửa</label>
            <Input
              type="time"
              value={openTime}
              onChange={(e) => setOpenTime(e.target.value)}
            />
          </div>
          <div>
            <label className="text-foreground mb-1.5 block text-sm font-medium">Giờ đóng cửa</label>
            <Input
              type="time"
              value={closeTime}
              onChange={(e) => setCloseTime(e.target.value)}
            />
          </div>
        </div>
        <div>
          <label className="text-foreground mb-1.5 block text-sm font-medium">
            Ghi chú (tuỳ chọn)
          </label>
          <Input
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="VD: Áp dụng từ tuần tới"
          />
        </div>
        <Button onClick={submit} disabled={m.requestHoursChange.isPending} className="w-full">
          {m.requestHoursChange.isPending
            ? <Loader2 className="h-4 w-4 animate-spin" />
            : 'Gửi yêu cầu'}
        </Button>
      </CardContent>
    </Card>
  );
}
