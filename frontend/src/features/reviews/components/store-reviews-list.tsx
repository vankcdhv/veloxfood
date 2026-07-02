'use client';

import { useState } from 'react';
import { Loader2, MessageSquare, Star } from 'lucide-react';
import { toast } from 'sonner';
import { Card, CardContent } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/dialog';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useInfiniteScroll } from '@/shared/hooks/use-infinite-scroll';
import { useStoreReviews, useStoreRatingSummary, useReportReview } from '../hooks/use-reviews';
import { StarRatingDisplay } from './star-rating-input';
import type { Review } from '../types/review';

interface StoreReviewsListProps {
  storeId: string;
}

export function StoreReviewsList({ storeId }: StoreReviewsListProps) {
  // 0 = all; 1–5 filters server-side to that star.
  const [starFilter, setStarFilter] = useState(0);
  const {
    data,
    isLoading,
    isError,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useStoreReviews(storeId, starFilter);
  const sentinelRef = useInfiniteScroll(fetchNextPage, { enabled: !!hasNextPage && !isFetchingNextPage });
  const reviews = data?.pages.flatMap((p) => p.Items) ?? [];
  const total = data?.pages[0]?.Total ?? 0;

  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-24 rounded-xl" />
        ))}
      </div>
    );
  }

  if (isError) {
    return <p className="text-destructive text-sm text-center py-8">Không tải được đánh giá.</p>;
  }

  if (reviews.length === 0 && starFilter === 0) {
    return (
      <div className="flex flex-col items-center gap-2 py-12 text-muted-foreground">
        <MessageSquare className="h-8 w-8 opacity-30" />
        <p className="text-sm">Chưa có đánh giá nào.</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <RatingHistogram storeId={storeId} active={starFilter} onSelect={setStarFilter} />
      <p className="text-sm text-muted-foreground">
        {total} đánh giá{starFilter > 0 && ` ${starFilter}★`}
      </p>
      {reviews.length === 0 ? (
        <p className="text-muted-foreground py-6 text-center text-sm">
          Không có đánh giá {starFilter}★ nào.
        </p>
      ) : (
        reviews.map((review) => <ReviewCard key={review.ID} review={review} />)
      )}
      <div ref={sentinelRef} />
      {isFetchingNextPage && (
        <div className="flex justify-center py-2">
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
        </div>
      )}
    </div>
  );
}

// RatingHistogram shows the 5→1★ distribution as clickable bars that filter
// the list to one star (click again to clear).
function RatingHistogram({
  storeId,
  active,
  onSelect,
}: {
  storeId: string;
  active: number;
  onSelect: (star: number) => void;
}) {
  const { data: summary } = useStoreRatingSummary(storeId);
  const counts = summary?.Counts;
  if (!counts || !summary || summary.Count === 0) return null;
  const max = Math.max(...counts, 1);

  return (
    <div className="border-border space-y-1 rounded-xl border p-3">
      {[5, 4, 3, 2, 1].map((star) => {
        const count = counts[star - 1] ?? 0;
        const selected = active === star;
        return (
          <button
            key={star}
            onClick={() => onSelect(selected ? 0 : star)}
            aria-pressed={selected}
            aria-label={`Lọc đánh giá ${star} sao (${count})`}
            className={`flex w-full items-center gap-2 rounded-md px-1.5 py-0.5 text-xs transition ${
              selected ? 'bg-primary/10' : 'hover:bg-muted/60'
            }`}
          >
            <span className="flex w-7 items-center gap-0.5 font-medium">
              {star}
              <Star className="h-3 w-3 fill-amber-400 text-amber-400" />
            </span>
            <span className="bg-muted h-2 flex-1 overflow-hidden rounded-full">
              <span
                className="bg-amber-400 block h-full rounded-full"
                style={{ width: `${(count / max) * 100}%` }}
              />
            </span>
            <span className="text-muted-foreground w-8 text-right tabular-nums">{count}</span>
          </button>
        );
      })}
    </div>
  );
}

export function ReviewCard({ review }: { review: Review }) {
  const [reportSent, setReportSent] = useState(false);
  const [zoomURL, setZoomURL] = useState<string | null>(null);
  const reportMutation = useReportReview();

  const handleReport = async () => {
    if (reportSent) return;
    try {
      await reportMutation.mutateAsync({ reviewId: review.ID, body: { reason: 'Nội dung không phù hợp' } });
      setReportSent(true);
      toast.success('Đã báo cáo đánh giá này');
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Không báo cáo được'));
    }
  };

  return (
    <Card>
      <CardContent className="py-4 space-y-2">
        <div className="flex items-start justify-between gap-2">
          <div className="space-y-0.5">
            <p className="text-sm font-medium">{review.ReviewerName ?? 'Người dùng'}</p>
            <StarRatingDisplay rating={review.Rating} size="sm" />
          </div>
          <span className="text-xs text-muted-foreground shrink-0">
            {new Date(review.CreatedAt).toLocaleDateString('vi-VN')}
          </span>
        </div>

        {review.Comment && (
          <p className="text-sm text-muted-foreground">{review.Comment}</p>
        )}

        {(review.PhotoURLs?.length ?? 0) > 0 && (
          <>
            <div className="flex flex-wrap gap-2">
              {review.PhotoURLs!.map((url) => (
                <button
                  key={url}
                  onClick={() => setZoomURL(url)}
                  aria-label="Xem ảnh đánh giá"
                  className="focus-visible:ring-ring rounded-lg focus-visible:ring-2 focus-visible:outline-none"
                >
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={url}
                    alt="Ảnh đánh giá"
                    className="border-border h-16 w-16 rounded-lg border object-cover"
                  />
                </button>
              ))}
            </div>
            <Dialog open={!!zoomURL} onOpenChange={(o) => !o && setZoomURL(null)}>
              <DialogContent className="max-w-2xl">
                <DialogHeader>
                  <DialogTitle>Ảnh đánh giá</DialogTitle>
                </DialogHeader>
                {zoomURL && (
                  /* eslint-disable-next-line @next/next/no-img-element */
                  <img src={zoomURL} alt="Ảnh đánh giá" className="max-h-[70vh] w-full rounded-lg object-contain" />
                )}
              </DialogContent>
            </Dialog>
          </>
        )}

        {review.Reply && (
          <div className="bg-muted rounded-md px-3 py-2 text-sm border-l-2 border-primary/40">
            <p className="text-xs font-medium text-primary mb-0.5">Phản hồi từ cửa hàng</p>
            <p className="text-muted-foreground">{review.Reply.Content}</p>
          </div>
        )}

        <button
          onClick={handleReport}
          disabled={reportSent || reportMutation.isPending}
          className="text-xs text-muted-foreground/60 hover:text-muted-foreground disabled:opacity-40 cursor-pointer"
        >
          {reportSent ? 'Đã báo cáo' : 'Báo cáo'}
        </button>
      </CardContent>
    </Card>
  );
}
