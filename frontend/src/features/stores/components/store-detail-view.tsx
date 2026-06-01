'use client';

import { useMemo, useState } from 'react';
import { MapPin, Phone, Truck, Star, Clock } from 'lucide-react';
import { toast } from 'sonner';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { Skeleton } from '@/shared/ui/skeleton';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useMyLocations } from '@/features/locations/hooks/use-locations';
import { formatRoomPath } from '@/features/locations/lib/format-room-path';
import { LocationPicker } from '@/features/locations/components/location-picker';
import { AddToCartButton } from '@/features/cart/components/add-to-cart-button';
import { StoreReviewsList } from '@/features/reviews/components/store-reviews-list';
import { browseStoreApi } from '../api/store-api';
import { useStore, useStoreMenu, useStoreSlots } from '../hooks/use-stores';
import { SaleStatusBadge } from './sale-status-badge';
import type { MenuCategory } from '../types/store';

type StoreTab = 'menu' | 'reviews';

interface StoreDetailViewProps {
  storeId: string;
}

// Local YYYY-MM-DD for a Date.
function isoDate(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

// Next 8 days as {value, label} for the session date picker.
function dateOptions(): { value: string; label: string }[] {
  const base = new Date();
  return Array.from({ length: 8 }, (_, i) => {
    const d = new Date(base);
    d.setDate(base.getDate() + i);
    const label = i === 0 ? 'Hôm nay' : i === 1 ? 'Ngày mai'
      : `${String(d.getDate()).padStart(2, '0')}/${String(d.getMonth() + 1).padStart(2, '0')}`;
    return { value: isoDate(d), label };
  });
}

export function StoreDetailView({ storeId }: StoreDetailViewProps) {
  const [tab, setTab] = useState<StoreTab>('menu');
  const [dates] = useState(dateOptions);
  const [selectedDate, setSelectedDate] = useState(() => isoDate(new Date()));
  const [selectedCutoff, setSelectedCutoff] = useState('');

  const { data: store, isLoading: storeLoading, isError: storeError } = useStore(storeId);
  const { data: menu, isLoading: menuLoading } = useStoreMenu(storeId);
  const { data: slots } = useStoreSlots(storeId, selectedDate);

  // Sort cutoffs by time so the session buttons read 11:00 → 17:30 → 22:00.
  const cutoffs = useMemo(
    () => [...(slots?.cutoffs ?? [])].sort((a, b) => a.CutoffTime.localeCompare(b.CutoffTime)),
    [slots],
  );
  const hasSlots = cutoffs.length > 0;

  // Derive the effective session (no effect needed): the user's pick if still
  // valid, otherwise the first available cutoff.
  const activeCutoff = useMemo(
    () => (cutoffs.some((c) => c.ID === selectedCutoff) ? selectedCutoff : (cutoffs[0]?.ID ?? '')),
    [cutoffs, selectedCutoff],
  );

  // menu_item_id → remaining quota for the selected session/date.
  const remainingByItem = useMemo(() => {
    const m = new Map<string, number>();
    for (const q of slots?.quotas ?? []) {
      if (q.CutoffID === activeCutoff) m.set(q.MenuItemID, q.Remaining);
    }
    return m;
  }, [slots, activeCutoff]);

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

      {/* Tab bar: Menu | Đánh giá */}
      <div className="bg-muted inline-flex rounded-lg p-1 gap-1" role="tablist">
        <button
          role="tab"
          aria-selected={tab === 'menu'}
          onClick={() => setTab('menu')}
          className={`rounded-md px-4 py-1.5 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
            tab === 'menu'
              ? 'bg-background text-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          Thực đơn
        </button>
        <button
          role="tab"
          aria-selected={tab === 'reviews'}
          onClick={() => setTab('reviews')}
          className={`flex items-center gap-1.5 rounded-md px-4 py-1.5 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
            tab === 'reviews'
              ? 'bg-background text-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          <Star className="h-3.5 w-3.5" />
          Đánh giá
        </button>
      </div>

      {/* Tab panels */}
      {tab === 'menu' && (
        <div className="space-y-6">
          {hasSlots && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <Clock className="h-4 w-4 text-primary" />
                  Chọn ca giao
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div>
                  <label className="text-foreground mb-1.5 block text-sm font-medium">Ngày</label>
                  <select
                    value={selectedDate}
                    onChange={(e) => setSelectedDate(e.target.value)}
                    className="border-input bg-background focus-visible:ring-ring h-9 w-full rounded-md border px-3 text-sm focus-visible:ring-2 focus-visible:outline-none"
                  >
                    {dates.map((d) => (
                      <option key={d.value} value={d.value}>{d.label}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="text-foreground mb-1.5 block text-sm font-medium">Ca</label>
                  <div className="flex flex-wrap gap-2">
                    {cutoffs.map((c) => (
                      <button
                        key={c.ID}
                        type="button"
                        onClick={() => setSelectedCutoff(c.ID)}
                        className={`rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors ${
                          activeCutoff === c.ID
                            ? 'border-primary bg-primary/10 text-primary'
                            : 'border-border text-muted-foreground hover:border-primary/50'
                        }`}
                      >
                        Ca {c.CutoffTime}
                      </button>
                    ))}
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
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
            <MenuCategorySection
              key={cat.Category.ID}
              cat={cat}
              slotMode={hasSlots}
              cutoffId={activeCutoff}
              date={selectedDate}
              remainingByItem={remainingByItem}
            />
          ))}
        </div>
      )}

      {tab === 'reviews' && (
        <StoreReviewsList storeId={storeId} />
      )}
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

interface MenuCategorySectionProps {
  cat: MenuCategory;
  slotMode: boolean;
  cutoffId: string;
  date: string;
  remainingByItem: Map<string, number>;
}

function MenuCategorySection({ cat, slotMode, cutoffId, date, remainingByItem }: MenuCategorySectionProps) {
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
          // In slot mode a missing quota row means the item isn't offered this session.
          const remaining = slotMode ? (remainingByItem.get(item.ID) ?? 0) : undefined;
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
                <div className="flex items-center justify-between gap-2 pt-0.5">
                  <div className="min-w-0">
                    <p className="text-primary font-semibold text-sm">
                      {item.Price.toLocaleString('vi-VN')}đ
                    </p>
                    {slotMode && (
                      <p className={`text-xs ${remaining! > 0 ? 'text-muted-foreground' : 'text-destructive'}`}>
                        {remaining! > 0 ? `Còn ${remaining} suất` : 'Hết suất'}
                      </p>
                    )}
                  </div>
                  <AddToCartButton
                    item={item}
                    cutoffId={slotMode ? cutoffId : undefined}
                    date={slotMode ? date : undefined}
                    remaining={remaining}
                  />
                </div>
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
