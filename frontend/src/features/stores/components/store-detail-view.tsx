'use client';

import { useState } from 'react';
import { MapPin, Phone, Truck, Star, Clock, UtensilsCrossed } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { Skeleton } from '@/shared/ui/skeleton';
import { formatVnd } from '@/shared/lib/format-vnd';
import { useMyLocations } from '@/features/locations/hooks/use-locations';
import { formatRoomPath } from '@/features/locations/lib/format-room-path';
import { LocationPicker } from '@/features/locations/components/location-picker';
import type { LocationSelection } from '@/features/locations/components/location-picker';
import { AddToCartButton } from '@/features/cart/components/add-to-cart-button';
import { StoreReviewsList } from '@/features/reviews/components/store-reviews-list';
import { MenuItemDetailDialog } from '@/features/reviews/components/menu-item-detail-dialog';
import { StarRatingDisplay } from '@/features/reviews/components/star-rating-input';
import { useStoreRatingSummary, useItemRatingSummaries } from '@/features/reviews/hooks/use-reviews';
import type { ItemRatingSummary } from '@/features/reviews/types/review';
import { useStore, useStoreMenu, useShipFee } from '../hooks/use-stores';
import { SaleStatusBadge } from './sale-status-badge';
import type { MenuCategory } from '../types/store';

type StoreTab = 'menu' | 'reviews';

interface StoreDetailViewProps {
  storeId: string;
}

export function StoreDetailView({ storeId }: StoreDetailViewProps) {
  const [tab, setTab] = useState<StoreTab>('menu');

  const { data: store, isLoading: storeLoading, isError: storeError } = useStore(storeId);
  const { data: menu, isLoading: menuLoading } = useStoreMenu(storeId);
  const { data: rating } = useStoreRatingSummary(storeId);
  const { data: itemSummaries } = useItemRatingSummaries(storeId);

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
        {rating && rating.Count > 0 && (
          <div className="flex items-center gap-1.5 text-sm">
            <StarRatingDisplay rating={Math.round(rating.Avg)} size="sm" />
            <span className="font-medium">{rating.Avg.toFixed(1)}</span>
            <span className="text-muted-foreground">({rating.Count} đánh giá)</span>
          </div>
        )}
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
        {/* Operating hours + prep time */}
        <div className="flex flex-wrap gap-3 text-sm">
          {store.OpenTimeToday && store.CloseTimeToday && (
            <span className="flex items-center gap-1 text-muted-foreground">
              <Clock className="h-4 w-4 shrink-0" />
              {store.OpenNow
                ? <span className="text-green-600 font-medium">Đang mở cửa</span>
                : <span className="text-destructive font-medium">Đang đóng</span>
              }
              <span className="ml-1">
                {store.OpenTimeToday}–{store.CloseTimeToday}
              </span>
            </span>
          )}
          {typeof store.PrepMinutes === 'number' && (
            <span className="text-muted-foreground">
              Chuẩn bị ~{store.PrepMinutes}&apos;
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
              storeId={storeId}
              itemSummaries={itemSummaries}
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
  const [sel, setSel] = useState<LocationSelection | null>(null);
  const [pickerKey, setPickerKey] = useState(0);

  const savedLocations = useMyLocations();
  const hasSavedLocations = (savedLocations.data?.length ?? 0) > 0;

  const feeQuery = useShipFee(
    storeId,
    sel?.level ?? '',
    sel?.id ?? '',
  );

  const notServed = !!sel && feeQuery.isError;

  const handleSavedLocationChange = (locationId: string) => {
    const loc = savedLocations.data?.find((l) => l.LocationID === locationId);
    setSel(loc ? { level: loc.LocationLevel, id: loc.LocationID } : null);
    setPickerKey((k) => k + 1);
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
        {hasSavedLocations && (
          <div>
            <label className="text-foreground mb-1.5 block text-sm font-medium">
              Vị trí đã lưu
            </label>
            <select
              value={sel?.id ?? ''}
              onChange={(e) => handleSavedLocationChange(e.target.value)}
              className="border-input bg-background focus-visible:ring-ring h-9 w-full rounded-md border px-3 text-sm focus-visible:ring-2 focus-visible:outline-none disabled:opacity-50"
            >
              <option value="">— Chọn vị trí —</option>
              {savedLocations.data?.map((loc) => (
                <option key={loc.ID} value={loc.LocationID}>
                  {loc.Label ? `${loc.Label} – ${formatRoomPath(loc)}` : formatRoomPath(loc)}
                </option>
              ))}
            </select>
          </div>
        )}

        {!hasSavedLocations && (
          <LocationPicker
            key={pickerKey}
            onSelectLocation={setSel}
          />
        )}

        {hasSavedLocations && (
          <details className="text-sm">
            <summary className="text-muted-foreground cursor-pointer select-none">
              Chọn địa điểm khác…
            </summary>
            <div className="mt-2">
              <LocationPicker
                key={pickerKey}
                onSelectLocation={setSel}
              />
            </div>
          </details>
        )}

        {feeQuery.isFetching && (
          <p className="text-muted-foreground text-sm">Đang tra phí…</p>
        )}
        {notServed && !feeQuery.isFetching && (
          <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive font-medium">
            Shop không hỗ trợ giao tới đây
          </div>
        )}
        {feeQuery.data && !feeQuery.isFetching && (
          <div className="rounded-md bg-muted px-3 py-2 text-sm">
            <p className="font-medium">
              Phí giao:{' '}
              <span className="text-primary font-semibold">
                {feeQuery.data.unit_ship_fee.toLocaleString('vi-VN')}đ
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
  storeId: string;
  itemSummaries?: Map<string, ItemRatingSummary>;
}

function MenuCategorySection({ cat, storeId, itemSummaries }: MenuCategorySectionProps) {
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
              {item.ImageURL ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={item.ImageURL}
                  alt={item.Name}
                  className="h-16 w-16 shrink-0 rounded-lg object-cover"
                />
              ) : (
                <div className="bg-muted text-muted-foreground/40 flex h-16 w-16 shrink-0 items-center justify-center rounded-lg">
                  <UtensilsCrossed className="h-6 w-6" />
                </div>
              )}
              <div className="min-w-0 flex-1 space-y-1">
                {(() => {
                  const s = itemSummaries?.get(item.ID);
                  return (
                    <MenuItemDetailDialog
                      storeId={storeId}
                      item={item}
                      avg={s?.Avg ?? 0}
                      count={s?.Count ?? 0}
                    />
                  );
                })()}
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
                  <p className="text-primary font-semibold text-sm">
                    {formatVnd(item.Price)}
                  </p>
                  {item.Status === 'off' ? (
                    <Badge variant="outline" className="shrink-0 text-xs">Tạm hết</Badge>
                  ) : (
                    <AddToCartButton item={item} />
                  )}
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
