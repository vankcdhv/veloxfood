'use client';

import { useEffect, useMemo, useState } from 'react';
import { Loader2, Search, Store as StoreIcon, X } from 'lucide-react';
import { Skeleton } from '@/shared/ui/skeleton';
import { useInfiniteScroll } from '@/shared/hooks/use-infinite-scroll';
import { useBrowseStores } from '../hooks/use-stores';
import { StoreCard } from './store-card';

type ModeFilter = 'all' | 'open' | 'pickup';
type SortKey = 'featured' | 'name';

const FILTERS: { key: ModeFilter; label: string }[] = [
  { key: 'all', label: 'Tất cả' },
  { key: 'open', label: 'Đang mở' },
  { key: 'pickup', label: 'Tự đến lấy' },
];

const SORTS: { key: SortKey; label: string }[] = [
  { key: 'featured', label: 'Nổi bật' },
  { key: 'name', label: 'Tên A → Z' },
];

export function StoreListView({ initialCuisine }: { initialCuisine?: string }) {
  const [query, setQuery] = useState('');
  const [mode, setMode] = useState<ModeFilter>('all');
  const [sort, setSort] = useState<SortKey>('featured');
  // Cuisine filter seeded from the home quick-row (?cuisine=). Removable.
  const [cuisine, setCuisine] = useState(initialCuisine ?? '');

  // Debounce the typed query before it hits the server (name filter).
  const [debouncedQuery, setDebouncedQuery] = useState('');
  useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(query), 300);
    return () => clearTimeout(t);
  }, [query]);

  const { data, isLoading, isError, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useBrowseStores(debouncedQuery);

  const loadMoreRef = useInfiniteScroll(fetchNextPage, {
    enabled: !!hasNextPage && !isFetchingNextPage,
  });

  // Mode + cuisine refine the already-loaded pages client-side (OpenNow is
  // computed per-request and cuisine isn't a server filter yet), then sort.
  const filtered = useMemo(() => {
    const all = data?.pages.flatMap((p) => p.items) ?? [];
    const out = all.filter((s) => {
      if (mode === 'open' && !s.OpenNow) return false;
      if (mode === 'pickup' && !s.PickupEnabled) return false;
      if (cuisine && s.BusinessType !== cuisine) return false;
      return true;
    });
    if (sort === 'name') {
      out.sort((a, b) => a.Name.localeCompare(b.Name, 'vi'));
    } else {
      // Featured: open stores first (stable within group preserves name order).
      out.sort((a, b) => Number(b.OpenNow ?? false) - Number(a.OpenNow ?? false));
    }
    return out;
  }, [data, mode, cuisine, sort]);

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
        <div className="flex flex-wrap items-center gap-2">
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

          {/* Active cuisine filter (from home quick-row) — removable */}
          {cuisine && (
            <button
              type="button"
              onClick={() => setCuisine('')}
              className="border-primary bg-primary/10 text-primary inline-flex items-center gap-1 rounded-full border px-3 py-1 text-xs font-medium"
            >
              {cuisine}
              <X className="h-3 w-3" />
            </button>
          )}

          {/* Sort — pushed to the right */}
          <select
            value={sort}
            onChange={(e) => setSort(e.target.value as SortKey)}
            aria-label="Sắp xếp"
            className="border-input bg-background ml-auto h-8 rounded-lg border px-2 text-xs"
          >
            {SORTS.map((s) => (
              <option key={s.key} value={s.key}>
                {s.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {filtered.length === 0 ? (
        <div className="flex flex-col items-center gap-3 py-16 text-muted-foreground">
          <StoreIcon className="h-10 w-10 opacity-30" />
          <p className="text-sm">
            {(data?.pages[0]?.total ?? 0) > 0
              ? 'Không có cửa hàng nào khớp bộ lọc.'
              : 'Chưa có cửa hàng nào.'}
          </p>
        </div>
      ) : (
        <div className="grid gap-4 auto-rows-fr sm:grid-cols-2 lg:grid-cols-3">
          {filtered.map((store) => (
            <StoreCard key={store.ID} store={store} href={`/stores/${store.ID}`} />
          ))}
        </div>
      )}

      {/* Infinite-scroll sentinel + loading indicator */}
      {hasNextPage && (
        <div ref={loadMoreRef} className="flex justify-center py-6">
          {isFetchingNextPage && <Loader2 className="text-muted-foreground h-5 w-5 animate-spin" />}
        </div>
      )}
    </div>
  );
}
