'use client';

import { useState } from 'react';
import { Store as StoreIcon } from 'lucide-react';
import { RoleGuard } from '@/features/auth/components/role-guard';
import { Skeleton } from '@/shared/ui/skeleton';
import { useMyStores } from '../hooks/use-stores';
import { StoreCard } from './store-card';
import { VendorStoreProfilePanel } from './vendor-store-profile-panel';
import { VendorCategoryPanel } from './vendor-category-panel';
import { VendorMenuPanel } from './vendor-menu-panel';
import { VendorShipFeePanel } from './vendor-ship-fee-panel';
import { VendorHoursPanel } from './vendor-hours-panel';
import { VendorOptionPanel } from './vendor-option-panel';
import { VendorComboPanel } from './vendor-combo-panel';
import { VendorPromotionPanel } from '@/features/promotions/components/vendor-promotion-panel';
import { VendorOrderPanel } from '@/features/orders/components/vendor-order-panel';
import { VendorReviewPanel } from '@/features/reviews/components/vendor-review-panel';
import { VendorRevenuePanel } from './vendor-revenue-panel';
import type { Store } from '../types/store';

// Tab definitions for the management panel.
const TABS = [
  { key: 'profile', label: 'Thông tin' },
  { key: 'categories', label: 'Danh mục' },
  { key: 'menu', label: 'Thực đơn' },
  { key: 'options', label: 'Tuỳ chọn' },
  { key: 'combos', label: 'Combo' },
  { key: 'shipfee', label: 'Phí giao' },
  { key: 'hours', label: 'Giờ hoạt động' },
  { key: 'promotions', label: 'Khuyến mãi' },
  { key: 'orders', label: 'Đơn hàng' },
  { key: 'revenue', label: 'Doanh thu' },
  { key: 'reviews', label: 'Đánh giá' },
] as const;
type TabKey = (typeof TABS)[number]['key'];

export function VendorStoreManagementView() {
  return (
    <RoleGuard allow={(a) => a.isAuthenticated}>
      <main className="bg-background min-h-dvh px-4 py-10">
        <div className="mx-auto w-full max-w-4xl space-y-8">
          <h1 className="font-serif text-2xl font-bold">Quản lý cửa hàng</h1>
          <StoreManagementContent />
        </div>
      </main>
    </RoleGuard>
  );
}

function StoreManagementContent() {
  const { data: stores, isLoading, isError } = useMyStores();
  const [selected, setSelected] = useState<Store | null>(null);
  const [tab, setTab] = useState<TabKey>('profile');

  if (isLoading) {
    return (
      <div className="grid gap-3 sm:grid-cols-2">
        {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-32 rounded-xl" />)}
      </div>
    );
  }

  if (isError) {
    return <p className="text-destructive text-sm">Không tải được danh sách cửa hàng.</p>;
  }

  if (!stores?.length) {
    return (
      <div className="flex flex-col items-center gap-3 py-16 text-muted-foreground">
        <StoreIcon className="h-10 w-10 opacity-30" />
        <p className="text-sm">Bạn chưa sở hữu cửa hàng nào.</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Store picker */}
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {stores.map((store) => (
          <button
            key={store.ID}
            type="button"
            onClick={() => { setSelected(store); setTab('profile'); }}
            className="text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-xl"
          >
            <div className={selected?.ID === store.ID ? 'ring-2 ring-primary rounded-xl' : ''}>
              <StoreCard store={store} />
            </div>
          </button>
        ))}
      </div>

      {/* Management panel for selected store */}
      {selected && (
        <ManagementPanel
          store={selected}
          tab={tab}
          onTabChange={setTab}
        />
      )}
    </div>
  );
}

function ManagementPanel({
  store, tab, onTabChange,
}: {
  store: Store;
  tab: TabKey;
  onTabChange: (t: TabKey) => void;
}) {
  return (
    <div className="space-y-4">
      {/* Tab bar */}
      <div className="bg-muted inline-flex rounded-lg p-1 flex-wrap gap-1" role="tablist">
        {TABS.map((t) => (
          <button
            key={t.key}
            role="tab"
            aria-selected={tab === t.key}
            onClick={() => onTabChange(t.key)}
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

      {/* Panel content */}
      {tab === 'profile' && <VendorStoreProfilePanel store={store} />}
      {tab === 'categories' && <VendorCategoryPanel storeId={store.ID} />}
      {tab === 'menu' && <VendorMenuPanel storeId={store.ID} />}
      {tab === 'options' && <VendorOptionPanel storeId={store.ID} />}
      {tab === 'combos' && <VendorComboPanel storeId={store.ID} />}
      {tab === 'shipfee' && <VendorShipFeePanel storeId={store.ID} />}
      {tab === 'hours' && <VendorHoursPanel storeId={store.ID} />}
      {tab === 'promotions' && <VendorPromotionPanel storeId={store.ID} />}
      {tab === 'orders' && <VendorOrderPanel storeId={store.ID} />}
      {tab === 'revenue' && <VendorRevenuePanel storeId={store.ID} />}
      {tab === 'reviews' && <VendorReviewPanel storeId={store.ID} />}
    </div>
  );
}
