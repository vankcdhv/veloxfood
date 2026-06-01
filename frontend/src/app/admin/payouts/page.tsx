'use client';

import { useState } from 'react';
import { AdminTopbar } from '@/widgets/admin-topbar/admin-topbar';
import { StoreSelectorForPayouts } from '@/features/payouts/components/store-selector-for-payouts';
import { SettlementPreviewPanel } from '@/features/payouts/components/settlement-preview-panel';
import { PayoutBatchHistoryTable } from '@/features/payouts/components/payout-batch-history-table';
import type { Store } from '@/features/stores/types/store';

export default function AdminPayoutsPage() {
  const [selected, setSelected] = useState<Store | null>(null);

  return (
    <>
      <AdminTopbar
        title="Chi trả cửa hàng"
        description="Xem số dư, tạo đợt chi trả và thực hiện thanh toán cho từng cửa hàng"
      />
      <div className="p-4 sm:p-6 lg:p-8">
        <div className="grid gap-6 lg:grid-cols-[280px_1fr]">
          {/* Left: store picker */}
          <StoreSelectorForPayouts
            selectedStoreId={selected?.ID ?? null}
            onSelect={setSelected}
          />

          {/* Right: settlement + batch history */}
          {selected ? (
            <div className="space-y-6">
              <SettlementPreviewPanel storeId={selected.ID} storeName={selected.Name} />
              <PayoutBatchHistoryTable storeId={selected.ID} />
            </div>
          ) : (
            <div className="flex items-center justify-center text-muted-foreground text-sm py-20">
              Chọn một cửa hàng để xem thông tin chi trả.
            </div>
          )}
        </div>
      </div>
    </>
  );
}
