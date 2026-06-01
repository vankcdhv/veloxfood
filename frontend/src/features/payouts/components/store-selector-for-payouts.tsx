'use client';

import { Store as StoreIcon } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { useAdminStores } from '@/features/stores/hooks/use-stores';
import type { Store } from '@/features/stores/types/store';

interface Props {
  selectedStoreId: string | null;
  onSelect: (store: Store) => void;
}

export function StoreSelectorForPayouts({ selectedStoreId, onSelect }: Props) {
  const { data: stores, isLoading, isError } = useAdminStores();

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm flex items-center gap-2">
          <StoreIcon className="h-4 w-4" />
          Chọn cửa hàng
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-1.5">
        {isLoading && (
          <div className="space-y-1.5">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-10 rounded-md" />
            ))}
          </div>
        )}

        {isError && (
          <p className="text-destructive text-sm">Không tải được danh sách cửa hàng.</p>
        )}

        {!isLoading && !isError && (!stores || stores.length === 0) && (
          <p className="text-muted-foreground text-sm">Chưa có cửa hàng nào.</p>
        )}

        {stores?.map((store) => (
          <button
            key={store.ID}
            type="button"
            onClick={() => onSelect(store)}
            className={`w-full text-left rounded-md border px-3 py-2 text-sm transition-colors hover:bg-accent/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
              selectedStoreId === store.ID
                ? 'border-primary/40 bg-primary/5 font-medium'
                : 'border-border'
            }`}
          >
            {store.Name}
            {store.BusinessType && (
              <span className="text-muted-foreground ml-1.5 text-xs">· {store.BusinessType}</span>
            )}
          </button>
        ))}
      </CardContent>
    </Card>
  );
}
