'use client';

import { useState } from 'react';
import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useVendorStoreMutations } from '../hooks/use-stores';

// Vietnamese weekday short labels: index 0=CN … 6=T7
const WEEKDAY_LABELS = ['CN', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7'];

interface WeekdayRow {
  enabled: boolean;
  open_time: string;
  close_time: string;
}

function makeDefaultRows(): WeekdayRow[] {
  return Array.from({ length: 7 }, () => ({ enabled: false, open_time: '08:00', close_time: '21:00' }));
}

export function VendorHoursSubmitForm({ storeId }: { storeId: string }) {
  const m = useVendorStoreMutations(storeId);
  const [rows, setRows] = useState<WeekdayRow[]>(makeDefaultRows);
  const [note, setNote] = useState('');

  const toggleDay = (i: number) => {
    setRows((prev) => prev.map((r, idx) => idx === i ? { ...r, enabled: !r.enabled } : r));
  };

  const updateRow = (i: number, field: 'open_time' | 'close_time', val: string) => {
    setRows((prev) => prev.map((r, idx) => idx === i ? { ...r, [field]: val } : r));
  };

  const submit = () => {
    const operating_hours = rows
      .map((r, weekday) => ({ weekday, ...r }))
      .filter((r) => r.enabled)
      .map(({ weekday, open_time, close_time }) => ({ weekday, open_time, close_time }));

    if (operating_hours.length === 0) {
      toast.error('Vui lòng chọn ít nhất một ngày hoạt động.');
      return;
    }

    m.requestHoursChange.mutate(
      { operating_hours, note: note.trim() },
      {
        onSuccess: () => {
          toast.success('Đã gửi yêu cầu — chờ admin duyệt.');
          setRows(makeDefaultRows());
          setNote('');
        },
        onError: (e) => toast.error(getApiErrorMessage(e, 'Gửi yêu cầu thất bại')),
      },
    );
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Đề xuất thay đổi giờ</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Weekday rows */}
        <div className="space-y-2">
          {rows.map((row, i) => (
            <div key={i} className="flex items-center gap-3 flex-wrap">
              <label className="flex items-center gap-1.5 cursor-pointer min-w-[56px]">
                <input
                  type="checkbox"
                  checked={row.enabled}
                  onChange={() => toggleDay(i)}
                  className="h-4 w-4 rounded border-border accent-primary"
                />
                <span className="text-sm font-medium">{WEEKDAY_LABELS[i]}</span>
              </label>
              {row.enabled ? (
                <div className="flex items-center gap-2">
                  <Input
                    type="time"
                    value={row.open_time}
                    onChange={(e) => updateRow(i, 'open_time', e.target.value)}
                    className="h-8 w-28 text-sm"
                  />
                  <span className="text-muted-foreground text-sm">–</span>
                  <Input
                    type="time"
                    value={row.close_time}
                    onChange={(e) => updateRow(i, 'close_time', e.target.value)}
                    className="h-8 w-28 text-sm"
                  />
                </div>
              ) : (
                <span className="text-muted-foreground text-xs">Đóng cửa</span>
              )}
            </div>
          ))}
        </div>

        {/* Note */}
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
