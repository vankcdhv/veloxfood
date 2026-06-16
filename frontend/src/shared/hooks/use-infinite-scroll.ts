'use client';

import { useEffect, useRef } from 'react';

interface Options {
  // When false the observer is detached (e.g. no more pages, or loading).
  enabled?: boolean;
  rootMargin?: string;
}

// useInfiniteScroll returns a ref to attach to a sentinel element near the end
// of a list. When the sentinel scrolls into view, onLoadMore() fires — pair it
// with useInfiniteQuery's fetchNextPage (which is referentially stable).
export function useInfiniteScroll<T extends HTMLElement = HTMLDivElement>(
  onLoadMore: () => void,
  { enabled = true, rootMargin = '200px' }: Options = {},
) {
  const ref = useRef<T | null>(null);

  useEffect(() => {
    const node = ref.current;
    if (!node || !enabled) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) onLoadMore();
      },
      { rootMargin },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [enabled, rootMargin, onLoadMore]);

  return ref;
}
