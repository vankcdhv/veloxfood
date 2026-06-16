'use client';

import { useState } from 'react';

// usePagedState owns a 1-indexed page that auto-resets to 1 whenever resetKey
// changes (a filter, tab, or selected-store switch). It uses the
// adjust-state-during-render pattern, so it never triggers a setState-in-effect
// cascade. Pass a primitive resetKey (combine multiple filters into one string).
export function usePagedState(resetKey: unknown): [number, (page: number) => void] {
  const [page, setPage] = useState(1);
  const [prevKey, setPrevKey] = useState(resetKey);
  if (resetKey !== prevKey) {
    setPrevKey(resetKey);
    setPage(1);
  }
  return [page, setPage];
}
