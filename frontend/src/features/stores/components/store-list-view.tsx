'use client';

import { Store as StoreIcon } from 'lucide-react';
import { Skeleton } from '@/shared/ui/skeleton';
import { useStores } from '../hooks/use-stores';
import { StoreCard } from './store-card';

export function StoreListView() {
  const { data, isLoading, isError } = useStores();

  if (isLoading) {
    return (
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-40 rounded-xl" />
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <p className="text-destructive text-center py-12 text-sm">
        Không tải được danh sách cửa hàng. Vui lòng thử lại.
      </p>
    );
  }

  if (!data?.length) {
    return (
      <div className="flex flex-col items-center gap-3 py-16 text-muted-foreground">
        <StoreIcon className="h-10 w-10 opacity-30" />
        <p className="text-sm">Chưa có cửa hàng nào.</p>
      </div>
    );
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {data.map((store) => (
        <StoreCard key={store.ID} store={store} href={`/stores/${store.ID}`} />
      ))}
    </div>
  );
}
