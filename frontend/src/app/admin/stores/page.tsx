'use client';

import { useState } from 'react';
import { AdminTopbar } from '@/widgets/admin-topbar/admin-topbar';
import { AdminStoreList, AdminStoreDetailCard } from '@/features/stores/components/admin-store-list';
import { AdminHoursChangeReview } from '@/features/stores/components/admin-hours-change-review';
import type { Store } from '@/features/stores/types/store';

export default function AdminStoresPage() {
  const [selected, setSelected] = useState<Store | null>(null);

  return (
    <>
      <AdminTopbar
        title="Quản lý cửa hàng"
        description="Tạo cửa hàng, xem thông tin và duyệt yêu cầu thay đổi giờ"
      />
      <div className="p-4 sm:p-6 lg:p-8">
        <div className="grid gap-6 lg:grid-cols-[320px_1fr]">
          {/* Left: store list */}
          <AdminStoreList
            onSelectStore={setSelected}
            selectedStoreId={selected?.ID}
          />

          {/* Right: selected store detail + hours-change review */}
          {selected ? (
            <div className="space-y-6">
              <AdminStoreDetailCard store={selected} />
              <div>
                <h3 className="font-semibold text-sm mb-3">Yêu cầu thay đổi giờ hoạt động</h3>
                <AdminHoursChangeReview storeId={selected.ID} />
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-center text-muted-foreground text-sm py-20">
              Chọn một cửa hàng để xem chi tiết.
            </div>
          )}
        </div>
      </div>
    </>
  );
}
