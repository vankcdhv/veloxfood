'use client';

import { useState } from 'react';
import { MapPin, Phone, Truck } from 'lucide-react';
import { toast } from 'sonner';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { Skeleton } from '@/shared/ui/skeleton';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useMyLocations } from '@/features/locations/hooks/use-locations';
import { formatRoomPath } from '@/features/locations/lib/format-room-path';
import { LocationPicker } from '@/features/locations/components/location-picker';
import { browseStoreApi } from '../api/store-api';
import { useStore, useStoreMenu } from '../hooks/use-stores';
import { SaleStatusBadge } from './sale-status-badge';
import type { MenuCategory } from '../types/store';

interface StoreDetailViewProps {
  storeId: string;
}

export function StoreDetailView({ storeId }: StoreDetailViewProps) {
  const { data: store, isLoading: storeLoading, isError: storeError } = useStore(storeId);
  const { data: menu, isLoading: menuLoading } = useStoreMenu(storeId);

  if (storeLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-24 rounded-xl" />
        <Skeleton className="h-64 rounded-xl" />
      </div>
    );
  }

  if (storeError || !store) {
    return (
      <p className="text-destructive text-center py-12 text-sm">
        Không tải được thông tin cửa hàng.
      </p>
    );
  }

  return (
    <div className="space-y-6">
      {/* Store header */}
      <div className="space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          <h1 className="font-serif text-2xl font-bold">{store.Name}</h1>
          <SaleStatusBadge status={store.SaleStatus} />
          {store.PickupEnabled && (
            <Badge variant="secondary">Tự lấy</Badge>
          )}
        </div>
        {store.BusinessType && (
          <p className="text-muted-foreground text-sm">{store.BusinessType}</p>
        )}
        <div className="flex flex-wrap gap-4 text-sm text-muted-foreground">
          {store.Address && (
            <span className="flex items-center gap-1">
              <MapPin className="h-4 w-4" />
              {store.Address}
            </span>
          )}
          {store.Phone && (
            <span className="flex items-center gap-1">
              <Phone className="h-4 w-4" />
              {store.Phone}
            </span>
          )}
        </div>
      </div>

      {/* Ship fee lookup */}
      <ShipFeeLookup storeId={storeId} />

      {/* Menu */}
      <div className="space-y-6">
        <h2 className="font-serif text-xl font-semibold">Thực đơn</h2>
        {menuLoading && (
          <div className="space-y-4">
            <Skeleton className="h-8 w-40" />
            <Skeleton className="h-48 rounded-xl" />
          </div>
        )}
        {!menuLoading && (!menu || menu.length === 0) && (
          <p className="text-muted-foreground text-sm">Chưa có món nào.</p>
        )}
        {menu?.map((cat) => (
          <MenuCategorySection key={cat.Category.ID} cat={cat} />
        ))}
      </div>
    </div>
  );
}

function ShipFeeLookup({ storeId }: { storeId: string }) {
  const [roomId, setRoomId] = useState('');
  // Human-readable label for the chosen room (from saved locations or picker).
  const [roomLabel, setRoomLabel] = useState('');
  const [fee, setFee] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);

  const savedLocations = useMyLocations();
  const hasSavedLocations = (savedLocations.data?.length ?? 0) > 0;

  const lookupFee = async (id: string) => {
    if (!id) return;
    setLoading(true);
    setFee(null);
    try {
      const result = await browseStoreApi.shipFee(storeId, id);
      setFee(result.unit_ship_fee);
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Không tra được phí giao hàng'));
    } finally {
      setLoading(false);
    }
  };

  const handleSavedLocationChange = (id: string) => {
    setRoomId(id);
    setFee(null);
    if (!id) { setRoomLabel(''); return; }
    const loc = savedLocations.data?.find((l) => l.RoomID === id);
    setRoomLabel(loc ? formatRoomPath(loc) : id);
    lookupFee(id);
  };

  const handlePickerSelect = (id: string) => {
    setRoomId(id);
    setFee(null);
    // Label will be resolved by the picker's own display; we just store the id.
    // Auto-lookup once a room is chosen.
    if (id) lookupFee(id);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <Truck className="h-4 w-4 text-primary" />
          Tra phí giao hàng
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {/* Primary: saved-locations dropdown when the customer has them */}
        {hasSavedLocations && (
          <div>
            <label className="text-foreground mb-1.5 block text-sm font-medium">
              Vị trí đã lưu
            </label>
            <select
              value={roomId}
              onChange={(e) => handleSavedLocationChange(e.target.value)}
              className="border-input bg-background focus-visible:ring-ring h-9 w-full rounded-md border px-3 text-sm focus-visible:ring-2 focus-visible:outline-none disabled:opacity-50"
            >
              <option value="">— Chọn vị trí —</option>
              {savedLocations.data?.map((loc) => (
                <option key={loc.ID} value={loc.RoomID}>
                  {loc.Label ? `${loc.Label} – ${formatRoomPath(loc)}` : formatRoomPath(loc)}
                </option>
              ))}
            </select>
          </div>
        )}

        {/* Fallback / secondary: cascading picker for ad-hoc room selection */}
        {!hasSavedLocations && (
          <LocationPicker
            level="room"
            value={roomId}
            onSelect={handlePickerSelect}
          />
        )}

        {/* When saved locations exist, also offer the picker for ad-hoc rooms */}
        {hasSavedLocations && (
          <details className="text-sm">
            <summary className="text-muted-foreground cursor-pointer select-none">
              Chọn phòng khác…
            </summary>
            <div className="mt-2">
              <LocationPicker
                level="room"
                value={roomId}
                onSelect={handlePickerSelect}
              />
            </div>
          </details>
        )}

        {loading && (
          <p className="text-muted-foreground text-sm">Đang tra phí…</p>
        )}
        {fee !== null && !loading && (
          <div className="rounded-md bg-muted px-3 py-2 text-sm">
            {roomLabel && (
              <p className="text-muted-foreground text-xs mb-0.5 truncate">{roomLabel}</p>
            )}
            <p className="font-medium">
              Phí giao:{' '}
              <span className="text-primary font-semibold">
                {fee.toLocaleString('vi-VN')}đ
              </span>
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function MenuCategorySection({ cat }: { cat: MenuCategory }) {
  return (
    <div className="space-y-3">
      <h3 className="font-semibold text-base border-b border-border pb-1">{cat.Category.Name}</h3>
      {cat.Items.length === 0 && (
        <p className="text-muted-foreground text-sm pl-1">Không có món nào trong danh mục này.</p>
      )}
      <div className="grid gap-3 sm:grid-cols-2">
        {cat.Items.map((item) => {
          const tags = item.Tags
            ? item.Tags.split(',').map((t) => t.trim()).filter(Boolean)
            : [];
          return (
            <div
              key={item.ID}
              className="flex items-start gap-3 rounded-lg border border-border p-3"
            >
              {item.ImageURL && (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={item.ImageURL}
                  alt={item.Name}
                  className="h-16 w-16 shrink-0 rounded-lg object-cover"
                />
              )}
              <div className="min-w-0 flex-1 space-y-1">
                <p className="font-medium text-sm leading-snug">{item.Name}</p>
                {item.Description && (
                  <p className="text-muted-foreground text-xs line-clamp-2">{item.Description}</p>
                )}
                {tags.length > 0 && (
                  <div className="flex flex-wrap gap-1">
                    {tags.map((tag) => (
                      <Badge key={tag} variant="secondary" className="text-xs px-1.5 py-0">
                        {tag}
                      </Badge>
                    ))}
                  </div>
                )}
                <p className="text-primary font-semibold text-sm">
                  {item.Price.toLocaleString('vi-VN')}đ
                </p>
              </div>
              {item.Status === 'off' && (
                <Badge variant="outline" className="shrink-0 text-xs">Hết hàng</Badge>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
