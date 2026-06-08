'use client';

import { useState } from 'react';
import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { PhotoUpload } from '@/shared/ui/photo-upload';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useVendorStoreMutations } from '../hooks/use-stores';
import { SaleStatusBadge } from './sale-status-badge';
import type { SaleStatus, Store } from '../types/store';

// Clamp prep minutes: must be a non-negative integer.
function parsePrepMinutes(raw: string): number {
  const n = parseInt(raw, 10);
  return Number.isNaN(n) || n < 0 ? 0 : n;
}

const SALE_STATUSES: { value: SaleStatus; label: string }[] = [
  { value: 'OPEN', label: 'Mở cửa' },
  { value: 'CLOSED_TODAY', label: 'Đóng hôm nay' },
  { value: 'PAUSED', label: 'Tạm dừng' },
];

interface Props {
  store: Store;
}

export function VendorStoreProfilePanel({ store }: Props) {
  const m = useVendorStoreMutations(store.ID);
  const [name, setName] = useState(store.Name);
  const [address, setAddress] = useState(store.Address);
  const [phone, setPhone] = useState(store.Phone);
  const [businessType, setBusinessType] = useState(store.BusinessType);
  const [prepMinutesRaw, setPrepMinutesRaw] = useState(String(store.PrepMinutes ?? 15));
  const [avatarFile, setAvatarFile] = useState<File | null>(null);

  const onPickAvatar = (file: File | null) => {
    setAvatarFile(file);
    if (!file) return;
    m.uploadAvatar.mutate(file, {
      onSuccess: () => { toast.success('Đã cập nhật ảnh cửa hàng.'); setAvatarFile(null); },
      onError: (e) => toast.error(getApiErrorMessage(e, 'Tải ảnh thất bại')),
    });
  };

  const saveProfile = () => {
    m.updateProfile.mutate(
      { name, address, phone, business_type: businessType, prep_minutes: parsePrepMinutes(prepMinutesRaw) },
      {
        onSuccess: () => toast.success('Đã lưu thông tin cửa hàng.'),
        onError: (e) => toast.error(getApiErrorMessage(e, 'Lưu thất bại')),
      },
    );
  };

  const changeSaleStatus = (status: SaleStatus) => {
    m.setSaleStatus.mutate(status, {
      onSuccess: () => toast.success('Đã cập nhật trạng thái.'),
      onError: (e) => toast.error(getApiErrorMessage(e, 'Cập nhật thất bại')),
    });
  };

  const togglePickup = () => {
    m.setPickup.mutate(!store.PickupEnabled, {
      onSuccess: () => toast.success('Đã cập nhật tuỳ chọn tự lấy.'),
      onError: (e) => toast.error(getApiErrorMessage(e, 'Cập nhật thất bại')),
    });
  };

  return (
    <div className="space-y-4">
      {/* Profile edit */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Thông tin cửa hàng</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {/* Store avatar/logo — uploads immediately on pick */}
          <div className="flex items-start gap-4">
            {store.AvatarURL ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={store.AvatarURL} alt={store.Name} className="h-16 w-16 shrink-0 rounded-xl object-cover" />
            ) : (
              <div className="bg-muted text-muted-foreground/40 flex h-16 w-16 shrink-0 items-center justify-center rounded-xl text-xs">
                Logo
              </div>
            )}
            <div className="flex-1">
              <PhotoUpload
                label="Ảnh / logo cửa hàng"
                value={avatarFile}
                onChange={onPickAvatar}
              />
              {m.uploadAvatar.isPending && (
                <p className="text-muted-foreground mt-1 flex items-center gap-1 text-xs">
                  <Loader2 className="h-3 w-3 animate-spin" /> Đang tải ảnh…
                </p>
              )}
            </div>
          </div>

          <Field label="Tên cửa hàng" value={name} onChange={setName} />
          <Field label="Loại hình" value={businessType} onChange={setBusinessType} />
          <Field label="Địa chỉ" value={address} onChange={setAddress} />
          <Field label="Số điện thoại" value={phone} onChange={setPhone} />
          <div>
            <label className="text-foreground mb-1.5 block text-sm font-medium">
              Thời gian chuẩn bị (phút)
            </label>
            <Input
              type="number"
              min={0}
              value={prepMinutesRaw}
              onChange={(e) => setPrepMinutesRaw(e.target.value)}
              placeholder="VD: 15"
            />
            <p className="text-muted-foreground mt-1 text-xs">
              Khách chọn giờ nhận sớm nhất = giờ đặt + thời gian chuẩn bị, làm tròn lên 15&apos;.
            </p>
          </div>
          <Button onClick={saveProfile} disabled={m.updateProfile.isPending} className="w-full">
            {m.updateProfile.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Lưu thông tin'}
          </Button>
        </CardContent>
      </Card>

      {/* Sale status */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            Trạng thái bán hàng
            <SaleStatusBadge status={store.SaleStatus} />
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          {SALE_STATUSES.map((s) => (
            <Button
              key={s.value}
              size="sm"
              variant={store.SaleStatus === s.value ? 'default' : 'outline'}
              disabled={m.setSaleStatus.isPending}
              onClick={() => changeSaleStatus(s.value)}
            >
              {s.label}
            </Button>
          ))}
        </CardContent>
      </Card>

      {/* Pickup toggle */}
      <Card>
        <CardContent className="flex items-center justify-between p-4">
          <div>
            <p className="font-medium text-sm">Cho phép tự lấy</p>
            <p className="text-muted-foreground text-xs">Khách đến lấy tại quầy</p>
          </div>
          <Button
            size="sm"
            variant={store.PickupEnabled ? 'default' : 'outline'}
            disabled={m.setPickup.isPending}
            onClick={togglePickup}
          >
            {store.PickupEnabled ? 'Đang bật' : 'Đang tắt'}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

function Field({
  label, value, onChange,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <div>
      <label className="text-foreground mb-1.5 block text-sm font-medium">{label}</label>
      <Input value={value} onChange={(e) => onChange(e.target.value)} />
    </div>
  );
}
