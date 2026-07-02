'use client';

import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Clock, History, Search, UtensilsCrossed, X } from 'lucide-react';
import { useQueries } from '@tanstack/react-query';
import { Skeleton } from '@/shared/ui/skeleton';
import { ItemThumbnail } from '@/shared/ui/item-thumbnail';
import { InfiniteScrollSentinel } from '@/shared/ui/infinite-scroll-sentinel';
import { useInfiniteScroll } from '@/shared/hooks/use-infinite-scroll';
import { formatVnd } from '@/shared/lib/format-vnd';
import {
  addSearchHistory,
  getSearchHistory,
  removeSearchHistory,
} from '@/shared/lib/search-history';
import { ROUTES } from '@/shared/config/constants';
import { useMenuItemSearch } from '@/features/stores/hooks/use-stores';
import { AddToCartButton } from '@/features/cart/components/add-to-cart-button';
import { MenuItemDetailDialog } from '@/features/reviews/components/menu-item-detail-dialog';
import { reviewApi } from '@/features/reviews/api/review-api';
import { reviewKeys } from '@/features/reviews/hooks/use-reviews';
import type {
  MenuItem,
  MenuItemSearchResult,
  MenuItemSearchSort,
} from '@/features/stores/types/store';
import type { ItemRatingSummary } from '@/features/reviews/types/review';

// Price-range presets (VND). Kept coarse — campus meals cluster under 100k.
const PRICE_RANGES = [
  { key: 'all', label: 'Mọi giá', min: 0, max: 0 },
  { key: 'lt25', label: 'Dưới 25k', min: 0, max: 25_000 },
  { key: '25to50', label: '25k – 50k', min: 25_000, max: 50_000 },
  { key: 'gt50', label: 'Trên 50k', min: 50_000, max: 0 },
] as const;
type PriceRangeKey = (typeof PRICE_RANGES)[number]['key'];

const SORT_OPTIONS: { key: MenuItemSearchSort; label: string }[] = [
  { key: 'relevance', label: 'Liên quan nhất' },
  { key: 'price_asc', label: 'Giá tăng dần' },
  { key: 'price_desc', label: 'Giá giảm dần' },
];

// Map a search result to the MenuItem shape expected by reused components.
// Search only returns currently sellable items, so Status is always 'on'.
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

// Outer page shell — wraps inner client component in Suspense (required for
// useSearchParams under Next.js App Router).
export default function SearchPage() {
  return (
    <div className="mx-auto w-full max-w-3xl px-4 py-10">
      <div className="mb-6 flex items-center gap-3">
        <div className="bg-primary/10 text-primary flex h-11 w-11 items-center justify-center rounded-xl">
          <Search className="h-6 w-6" />
        </div>
        <div>
          <h1 className="font-serif text-2xl font-bold">Tìm món ăn</h1>
          <p className="text-muted-foreground text-sm">Tìm kiếm món ăn từ tất cả cửa hàng.</p>
        </div>
      </div>
      <Suspense fallback={<SearchSkeleton />}>
        <SearchContent />
      </Suspense>
    </div>
  );
}

function SearchContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const initialQ = searchParams.get('q') ?? '';

  const [inputValue, setInputValue] = useState(initialQ);
  const [debouncedQ, setDebouncedQ] = useState(initialQ);
  const [sort, setSort] = useState<MenuItemSearchSort>('relevance');
  const [priceKey, setPriceKey] = useState<PriceRangeKey>('all');
  const [openOnly, setOpenOnly] = useState(false);
  const [history, setHistory] = useState<string[]>([]);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const historyRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Autofocus on mount; history read client-side only (localStorage), deferred
  // a tick so hydration completes before the recent-search chips appear.
  useEffect(() => {
    inputRef.current?.focus();
    const t = setTimeout(() => setHistory(getSearchHistory()), 0);
    return () => {
      clearTimeout(t);
      if (historyRef.current) clearTimeout(historyRef.current);
    };
  }, []);

  const handleInput = useCallback(
    (value: string) => {
      setInputValue(value);
      if (debounceRef.current) clearTimeout(debounceRef.current);
      debounceRef.current = setTimeout(() => {
        setDebouncedQ(value);
        const params = new URLSearchParams(searchParams.toString());
        if (value.trim()) {
          params.set('q', value);
        } else {
          params.delete('q');
        }
        router.replace(`${ROUTES.search}?${params.toString()}`);
      }, 300);
      // Record the term once typing settles — avoids saving every keystroke.
      if (historyRef.current) clearTimeout(historyRef.current);
      if (value.trim().length >= 2) {
        historyRef.current = setTimeout(() => setHistory(addSearchHistory(value)), 1500);
      }
    },
    [router, searchParams],
  );

  const pickHistory = useCallback(
    (term: string) => {
      setInputValue(term);
      setDebouncedQ(term);
      setHistory(addSearchHistory(term));
      const params = new URLSearchParams(searchParams.toString());
      params.set('q', term);
      router.replace(`${ROUTES.search}?${params.toString()}`);
    },
    [router, searchParams],
  );

  const range = PRICE_RANGES.find((r) => r.key === priceKey) ?? PRICE_RANGES[0];
  const filters = useMemo(
    () => ({ sort, priceMin: range.min, priceMax: range.max }),
    [sort, range.min, range.max],
  );

  const { data, isLoading, isError, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useMenuItemSearch(debouncedQ, filters);
  const results = useMemo(() => {
    const rows = data?.pages.flatMap((p) => p.items) ?? [];
    // "Đang mở" narrows the loaded rows client-side (open state is computed
    // per store server-side but is not a SQL filter).
    return openOnly ? rows.filter((r) => r.OpenNow) : rows;
  }, [data, openOnly]);
  const total = data?.pages[0]?.total ?? 0;

  const loadMoreRef = useInfiniteScroll(fetchNextPage, {
    enabled: !!hasNextPage && !isFetchingNextPage,
  });

  return (
    <div className="space-y-5">
      {/* Search input */}
      <div className="relative">
        <Search className="text-muted-foreground pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2" />
        <input
          ref={inputRef}
          type="search"
          value={inputValue}
          onChange={(e) => handleInput(e.target.value)}
          placeholder="Nhập tên món để tìm…"
          className="border-input bg-background focus-visible:ring-ring h-11 w-full rounded-xl border py-2 pl-10 pr-4 text-sm shadow-sm transition focus-visible:outline-none focus-visible:ring-2"
        />
      </div>

      {/* Recent searches — only when the box is empty */}
      {!inputValue.trim() && history.length > 0 && (
        <div className="space-y-2">
          <p className="text-muted-foreground flex items-center gap-1.5 text-xs font-medium">
            <History className="h-3.5 w-3.5" /> Tìm kiếm gần đây
          </p>
          <div className="flex flex-wrap gap-2">
            {history.map((term) => (
              <span
                key={term}
                className="border-border bg-muted/40 inline-flex items-center gap-1 rounded-full border py-1 pl-3 pr-1 text-sm"
              >
                <button onClick={() => pickHistory(term)} className="hover:text-primary">
                  {term}
                </button>
                <button
                  onClick={() => setHistory(removeSearchHistory(term))}
                  aria-label={`Xoá "${term}" khỏi lịch sử`}
                  className="text-muted-foreground hover:text-destructive rounded-full p-0.5"
                >
                  <X className="h-3 w-3" />
                </button>
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Filter / sort bar — only meaningful once there is a query */}
      {debouncedQ.trim() && (
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <select
            value={sort}
            onChange={(e) => setSort(e.target.value as MenuItemSearchSort)}
            aria-label="Sắp xếp kết quả"
            className="border-input bg-background focus-visible:ring-ring h-9 rounded-lg border px-2 focus-visible:outline-none focus-visible:ring-2"
          >
            {SORT_OPTIONS.map((o) => (
              <option key={o.key} value={o.key}>
                {o.label}
              </option>
            ))}
          </select>
          <select
            value={priceKey}
            onChange={(e) => setPriceKey(e.target.value as PriceRangeKey)}
            aria-label="Lọc theo khoảng giá"
            className="border-input bg-background focus-visible:ring-ring h-9 rounded-lg border px-2 focus-visible:outline-none focus-visible:ring-2"
          >
            {PRICE_RANGES.map((r) => (
              <option key={r.key} value={r.key}>
                {r.label}
              </option>
            ))}
          </select>
          <button
            onClick={() => setOpenOnly((v) => !v)}
            aria-pressed={openOnly}
            className={`focus-visible:ring-ring inline-flex h-9 items-center gap-1.5 rounded-lg border px-3 transition focus-visible:outline-none focus-visible:ring-2 ${
              openOnly
                ? 'border-green-600 bg-green-50 text-green-700 dark:bg-green-950/30 dark:text-green-400'
                : 'border-input bg-background text-muted-foreground hover:text-foreground'
            }`}
          >
            <Clock className="h-3.5 w-3.5" /> Đang mở
          </button>
        </div>
      )}

      {/* Result area */}
      <SearchResults
        q={debouncedQ}
        results={results}
        total={total}
        isLoading={isLoading}
        isError={isError}
        hasNextPage={!!hasNextPage}
        isFetchingNextPage={isFetchingNextPage}
        loadMoreRef={loadMoreRef}
      />
    </div>
  );
}

interface SearchResultsProps {
  q: string;
  results: MenuItemSearchResult[];
  total: number;
  isLoading: boolean;
  isError: boolean;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  loadMoreRef: React.RefObject<HTMLDivElement | null>;
}

function SearchResults({
  q,
  results,
  total,
  isLoading,
  isError,
  hasNextPage,
  isFetchingNextPage,
  loadMoreRef,
}: SearchResultsProps) {
  // Collect distinct store IDs from current results so we can batch-fetch
  // per-item rating summaries across all stores in parallel.
  const storeIds = useMemo(
    () => [...new Set(results.map((r) => r.StoreID))],
    [results],
  );

  // Fire one query per distinct store. useQueries with an empty array is safe
  // and returns [] — hooks are always called unconditionally.
  const ratingQueries = useQueries({
    queries: storeIds.map((sid) => ({
      queryKey: reviewKeys.itemSummaries(sid),
      // Return a Map to match useItemRatingSummaries (shares this queryKey; the
      // cache value must be the same shape or store-detail's .get() breaks).
      queryFn: async () =>
        new Map((await reviewApi.itemSummaries(sid) ?? []).map((s) => [s.TargetID, s])),
      enabled: storeIds.length > 0,
    })),
  });

  // Merge all per-store summary maps into a single map keyed by menu-item ID.
  const ratingsMap = useMemo<Map<string, ItemRatingSummary>>(() => {
    const map = new Map<string, ItemRatingSummary>();
    ratingQueries.forEach((q) => {
      (q.data as Map<string, ItemRatingSummary> | undefined)?.forEach((s) => map.set(s.TargetID, s));
    });
    return map;
  }, [ratingQueries]);

  // --- Early returns (after all hooks) ---

  // Empty query — prompt
  if (!q.trim()) {
    return (
      <p className="text-muted-foreground py-10 text-center text-sm">
        Nhập tên món để tìm kiếm trên tất cả cửa hàng.
      </p>
    );
  }

  // Loading state — skeleton rows
  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="flex items-center gap-3 rounded-xl border border-border p-3">
            <Skeleton className="h-16 w-16 shrink-0 rounded-lg" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-4 w-2/3" />
              <Skeleton className="h-3 w-1/2" />
              <Skeleton className="h-3 w-1/4" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  // Error state
  if (isError) {
    return (
      <p className="text-destructive py-10 text-center text-sm">
        Đã có lỗi khi tìm kiếm. Vui lòng thử lại.
      </p>
    );
  }

  // No results
  if (results.length === 0) {
    return (
      <div className="py-10 text-center">
        <UtensilsCrossed className="text-muted-foreground/40 mx-auto mb-3 h-10 w-10" />
        <p className="text-muted-foreground text-sm">
          Không tìm thấy món nào khớp với &ldquo;{q}&rdquo;.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <p className="text-muted-foreground text-xs">
        {total} kết quả cho &ldquo;{q}&rdquo;
      </p>
      {results.map((r) => {
        const mapped = toMenuItem(r);
        const summary = ratingsMap.get(r.ID);
        return (
          <div
            key={r.ID}
            className="flex items-start gap-3 rounded-xl border border-border p-3 transition hover:border-orange-500/50 hover:bg-orange-50/50 dark:hover:bg-orange-950/20"
          >
            <ItemThumbnail src={r.ImageURL} alt={r.Name} />

            {/* Info */}
            <div className="min-w-0 flex-1 space-y-1">
              {/* Name + star rating line; opens detail dialog with reviews on click */}
              <MenuItemDetailDialog
                storeId={r.StoreID}
                item={mapped}
                avg={summary?.Avg ?? 0}
                count={summary?.Count ?? 0}
              />

              {/* Description */}
              {r.Description && (
                <p className="text-muted-foreground text-xs line-clamp-2">{r.Description}</p>
              )}

              {/* Price row: price left, add-to-cart right */}
              <div className="flex items-center justify-between gap-2 pt-0.5">
                <div>
                  <p className="text-primary font-semibold text-sm">{formatVnd(r.Price)}</p>
                  <p className="text-muted-foreground text-xs">
                    <Link
                      href={ROUTES.stores.detail(r.StoreID)}
                      className="hover:underline"
                      onClick={(e) => e.stopPropagation()}
                    >
                      {r.StoreName}
                    </Link>
                    <OpenNowHint open={r.OpenNow} />
                  </p>
                </div>
                <AddToCartButton item={mapped} />
              </div>
            </div>
          </div>
        );
      })}

      <InfiniteScrollSentinel
        loadMoreRef={loadMoreRef}
        hasNextPage={hasNextPage}
        isFetchingNextPage={isFetchingNextPage}
      />
    </div>
  );
}

// OpenNowHint reflects the server-computed open state (hours + sale status),
// which is what actually gates ordering.
function OpenNowHint({ open }: { open: boolean }) {
  return open ? (
    <span className="ml-1 text-green-600">· Đang mở</span>
  ) : (
    <span className="text-destructive ml-1">· Đang đóng cửa</span>
  );
}

function SearchSkeleton() {
  return (
    <div className="space-y-5">
      <Skeleton className="h-11 w-full rounded-xl" />
      <div className="space-y-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-[88px] rounded-xl" />
        ))}
      </div>
    </div>
  );
}
