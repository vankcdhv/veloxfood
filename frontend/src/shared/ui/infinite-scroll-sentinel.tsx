import { Loader2 } from 'lucide-react';

interface InfiniteScrollSentinelProps {
  // Ref from useInfiniteScroll — observed to trigger the next page fetch.
  loadMoreRef: React.RefObject<HTMLDivElement | null>;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
}

// InfiniteScrollSentinel is the standard end-of-list marker: invisible while
// idle, spinner while the next page loads. Render after the list items.
export function InfiniteScrollSentinel({
  loadMoreRef,
  hasNextPage,
  isFetchingNextPage,
}: InfiniteScrollSentinelProps) {
  if (!hasNextPage) return null;
  return (
    <div ref={loadMoreRef} className="flex justify-center py-6">
      {isFetchingNextPage && <Loader2 className="text-muted-foreground h-5 w-5 animate-spin" />}
    </div>
  );
}
