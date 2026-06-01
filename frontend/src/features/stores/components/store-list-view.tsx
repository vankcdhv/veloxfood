'use client';

import { useMemo, useState } from 'react';
import { Search, Store as StoreIcon } from 'lucide-react';
import { Skeleton } from '@/shared/ui/skeleton';
import { useStores } from '../hooks/use-stores';
import { StoreCard } from './store-card';

type ModeFilter = 'all' | 'open' | 'pickup';

const FILTERS: { key: ModeFilter; label: string }[] = [
  { key: 'all', label: 'Tất cả' },
  { key: 'open', label: 'Đang mở' },
  { key: 'pickup', label: 'Tự đến lấy' },
];

export function StoreListView() {
  const { data, isLoading, isError } = useStores();
  const [query, setQuery] = useState('');
  const [mode, setMode] = useState<ModeFilter>('all');

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return (data ?? []).filter((s) => {
      if (q && !`${s.Name} ${s.BusinessType ?? ''}`.toLowerCase().includes(q)) return false;
      if (mode === 'open' && s.SaleStatus !== 'OPEN') return false;
      if (mode === 'pickup' && !s.PickupEnabled) return false;
      return true;
    });
  }, [data, query, mode]);

  if (isLoading) {
    return (
      <div className="grid gap-4 auto-rows-fr sm:grid-cols-2 lg:grid-cols-3">
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

  return (
    <div className="space-y-5">
      {/* Search + filter chips */}
      <div className="space-y-3">
        <div className="relative">
          <Search className="text-muted-foreground absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2" />
          <input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Tìm cửa hàng hoặc loại món…"
            className="border-input bg-background focus-visible:ring-ring h-10 w-full rounded-lg border pl-9 pr-3 text-sm focus-visible:ring-2 focus-visible:outline-none"
          />
        </div>
        <div className="flex flex-wrap gap-2">
          {FILTERS.map((f) => (
            <button
              key={f.key}
              type="button"
              onClick={() => setMode(f.key)}
              className={`rounded-full border px-3 py-1 text-xs font-medium transition-colors ${
                mode === f.key
                  ? 'border-primary bg-primary/10 text-primary'
                  : 'border-border text-muted-foreground hover:border-primary/50'
              }`}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {filtered.length === 0 ? (
        <div className="flex flex-col items-center gap-3 py-16 text-muted-foreground">
          <StoreIcon className="h-10 w-10 opacity-30" />
          <p className="text-sm">
            {data?.length ? 'Không có cửa hàng nào khớp bộ lọc.' : 'Chưa có cửa hàng nào.'}
          </p>
        </div>
      ) : (
        <div className="grid gap-4 auto-rows-fr sm:grid-cols-2 lg:grid-cols-3">
          {filtered.map((store) => (
            <StoreCard key={store.ID} store={store} href={`/stores/${store.ID}`} />
          ))}
        </div>
      )}
    </div>
  );
}
