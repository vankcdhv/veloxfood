'use client';

import { useState } from 'react';
import { toast } from 'sonner';
import { MessageSquare } from 'lucide-react';
import { Button } from '@/shared/ui/button';
import { Card, CardContent } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useStoreReviews, useReplyReview } from '../hooks/use-reviews';
import { StarRatingDisplay } from './star-rating-input';
import type { Review } from '../types/review';

interface VendorReviewPanelProps {
  storeId: string;
}

export function VendorReviewPanel({ storeId }: VendorReviewPanelProps) {
  const { data, isLoading, isError } = useStoreReviews(storeId);

  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-28 rounded-xl" />
        ))}
      </div>
    );
  }

  if (isError) {
    return <p className="text-destructive text-sm">Không tải được đánh giá.</p>;
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
        <VendorReviewCard key={review.ID} review={review} storeId={storeId} />
      ))}
    </div>
  );
}

function VendorReviewCard({ review, storeId }: { review: Review; storeId: string }) {
  const [replying, setReplying] = useState(false);
  const [replyText, setReplyText] = useState('');
  const replyMutation = useReplyReview(storeId);

  const handleReply = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!replyText.trim()) return;
    try {
      await replyMutation.mutateAsync({ reviewId: review.ID, body: { content: replyText.trim() } });
      toast.success('Đã phản hồi đánh giá');
      setReplying(false);
      setReplyText('');
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Không phản hồi được'));
    }
  };

  return (
    <Card>
      <CardContent className="py-4 space-y-3">
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

        {review.Reply ? (
          <div className="bg-muted rounded-md px-3 py-2 text-sm border-l-2 border-primary/40">
            <p className="text-xs font-medium text-primary mb-0.5">Phản hồi của bạn</p>
            <p className="text-muted-foreground">{review.Reply.Content}</p>
          </div>
        ) : (
          <>
            {!replying && (
              <Button variant="outline" size="sm" onClick={() => setReplying(true)}>
                <MessageSquare className="h-3.5 w-3.5 mr-1.5" />
                Phản hồi
              </Button>
            )}
            {replying && (
              <form onSubmit={handleReply} className="space-y-2">
                <textarea
                  value={replyText}
                  onChange={(e) => setReplyText(e.target.value)}
                  placeholder="Nhập phản hồi của bạn…"
                  rows={2}
                  disabled={replyMutation.isPending}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50 resize-none"
                />
                <div className="flex gap-2">
                  <Button type="submit" size="sm" disabled={replyMutation.isPending || !replyText.trim()}>
                    {replyMutation.isPending ? 'Đang gửi…' : 'Gửi'}
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => { setReplying(false); setReplyText(''); }}
                    disabled={replyMutation.isPending}
                  >
                    Hủy
                  </Button>
                </div>
              </form>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}
