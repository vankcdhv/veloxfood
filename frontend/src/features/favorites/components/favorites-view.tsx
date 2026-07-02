'use client';

import { useState } from 'react';
import Link from 'next/link';
import { Heart, Store as StoreIcon, UtensilsCrossed } from 'lucide-react';
import { Skeleton } from '@/shared/ui/skeleton';
import { ItemThumbnail } from '@/shared/ui/item-thumbnail';
import { formatVnd } from '@/shared/lib/format-vnd';
import { ROUTES } from '@/shared/config/constants';
import { RoleGuard } from '@/features/auth/components/role-guard';
import { StoreCard } from '@/features/stores/components/store-card';
import { AddToCartButton } from '@/features/cart/components/add-to-cart-button';
import type { MenuItem, MenuItemSearchResult } from '@/features/stores/types/store';
import { useFavoriteItems, useFavoriteStores } from '../hooks/use-favorites';
import { FavoriteButton } from './favorite-button';

type Tab = 'store' | 'item';

const TABS: { key: Tab; label: string }[] = [
  { key: 'store', label: 'Quán yêu thích' },
  { key: 'item', label: 'Món yêu thích' },
];

// FavoritesView renders the account page with bookmarked stores and dishes.
export function FavoritesView() {
  const [tab, setTab] = useState<Tab>('store');

  return (
    <RoleGuard allow={(a) => a.isAuthenticated}>
      <main className="bg-background min-h-dvh px-4 py-10">
        <div className="mx-auto w-full max-w-4xl space-y-6">
          <div className="flex items-center gap-3">
            <div className="bg-primary/10 text-primary flex h-11 w-11 items-center justify-center rounded-xl">
              <Heart className="h-6 w-6" />
            </div>
            <div>
              <h1 className="font-serif text-2xl font-bold">Yêu thích</h1>
              <p className="text-muted-foreground text-sm">Quán và món bạn đã lưu lại.</p>
            </div>
          </div>

          <div
            className="bg-muted inline-flex rounded-lg p-1"
            role="tablist"
            aria-label="Loại yêu thích"
          >
            {TABS.map((t) => (
              <button
                key={t.key}
                role="tab"
                aria-selected={tab === t.key}
                onClick={() => setTab(t.key)}
                className={`focus-visible:ring-ring rounded-md px-3 py-1.5 text-sm font-medium transition-colors focus-visible:ring-2 focus-visible:outline-none ${
                  tab === t.key
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                {t.label}
              </button>
            ))}
          </div>

          {tab === 'store' ? <FavoriteStoresTab /> : <FavoriteItemsTab />}
        </div>
      </main>
    </RoleGuard>
  );
}

function EmptyState({ icon: Icon, children }: { icon: typeof Heart; children: React.ReactNode }) {
  return (
    <div className="text-muted-foreground flex flex-col items-center gap-3 py-16 text-center">
      <Icon className="h-10 w-10 opacity-30" />
      <p className="text-sm">{children}</p>
      <Link href={ROUTES.stores.root} className="text-primary text-sm font-medium hover:underline">
        Khám phá cửa hàng →
      </Link>
    </div>
  );
}

function FavoriteStoresTab() {
  const { data, isLoading, isError } = useFavoriteStores();

  if (isLoading) {
    return (
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-40 rounded-xl" />
        ))}
      </div>
    );
  }
  if (isError) {
    return <p className="text-destructive py-10 text-center text-sm">Không tải được danh sách yêu thích.</p>;
  }
  if (!data || data.length === 0) {
    return <EmptyState icon={StoreIcon}>Bạn chưa lưu quán nào. Nhấn ♥ trên quán để lưu lại.</EmptyState>;
  }

  return (
    <div className="grid gap-4 auto-rows-fr sm:grid-cols-2 lg:grid-cols-3">
      {data.map((store) => (
        <StoreCard key={store.ID} store={store} href={ROUTES.stores.detail(store.ID)} />
      ))}
    </div>
  );
}

// Map a favorite item row to the MenuItem shape reused by cart components.
function toMenuItem(r: MenuItemSearchResult): MenuItem {
  return {
    ID: r.ID,
    StoreID: r.StoreID,
    CategoryID: '',
    Name: r.Name,
    Description: r.Description,
    Price: r.Price,
    ImageURL: r.ImageURL,
    Status: 'on',
    Tags: '',
  };
}

function FavoriteItemsTab() {
  const { data, isLoading, isError } = useFavoriteItems();

  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-[88px] rounded-xl" />
        ))}
      </div>
    );
  }
  if (isError) {
    return <p className="text-destructive py-10 text-center text-sm">Không tải được danh sách yêu thích.</p>;
  }
  if (!data || data.length === 0) {
    return (
      <EmptyState icon={UtensilsCrossed}>Bạn chưa lưu món nào. Nhấn ♥ trên món để lưu lại.</EmptyState>
    );
  }

  return (
    <div className="space-y-2">
      {data.map((r) => (
        <div
          key={r.ID}
          className="border-border flex items-start gap-3 rounded-xl border p-3 transition hover:border-orange-500/50"
        >
          <ItemThumbnail src={r.ImageURL} alt={r.Name} />
          <div className="min-w-0 flex-1 space-y-1">
            <div className="flex items-start justify-between gap-2">
              <p className="text-sm font-medium">{r.Name}</p>
              <FavoriteButton type="item" id={r.ID} name={r.Name} />
            </div>
            {r.Description && (
              <p className="text-muted-foreground line-clamp-2 text-xs">{r.Description}</p>
            )}
            <div className="flex items-center justify-between gap-2 pt-0.5">
              <div>
                <p className="text-primary text-sm font-semibold">{formatVnd(r.Price)}</p>
                <p className="text-muted-foreground text-xs">
                  <Link href={ROUTES.stores.detail(r.StoreID)} className="hover:underline">
                    {r.StoreName}
                  </Link>
                  {r.OpenNow ? (
                    <span className="ml-1 text-green-600">· Đang mở</span>
                  ) : (
                    <span className="text-destructive ml-1">· Đang đóng cửa</span>
                  )}
                </p>
              </div>
              <AddToCartButton item={toMenuItem(r)} />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
