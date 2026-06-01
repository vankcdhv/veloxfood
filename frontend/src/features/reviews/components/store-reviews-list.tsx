'use client';

import { useState } from 'react';
import { MessageSquare } from 'lucide-react';
import { toast } from 'sonner';
import { Card, CardContent } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useStoreReviews, useReportReview } from '../hooks/use-reviews';
import { StarRatingDisplay } from './star-rating-input';
import type { Review } from '../types/review';

interface StoreReviewsListProps {
  storeId: string;
}

export function StoreReviewsList({ storeId }: StoreReviewsListProps) {
  const { data, isLoading, isError } = useStoreReviews(storeId);

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

  const reviews = data?.Items ?? [];

  if (reviews.length === 0) {
    return (
      <div className="flex flex-col items-center gap-2 py-12 text-muted-foreground">
        <MessageSquare className="h-8 w-8 opacity-30" />
        <p className="text-sm">Chưa có đánh giá nào.</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <p className="text-sm text-muted-foreground">{data?.Total ?? reviews.length} đánh giá</p>
      {reviews.map((review) => (
        <ReviewCard key={review.ID} review={review} />
      ))}
    </div>
  );
}

function ReviewCard({ review }: { review: Review }) {
  const [reportSent, setReportSent] = useState(false);
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
