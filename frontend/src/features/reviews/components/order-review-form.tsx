'use client';

import { useState } from 'react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useCreateReview } from '../hooks/use-reviews';
import { StarRatingInput } from './star-rating-input';

interface OrderReviewFormProps {
  orderId: string;
  storeId: string;
  /** Called after a successful submission so the parent can hide the form. */
  onSubmitted?: () => void;
}

export function OrderReviewForm({ orderId, storeId, onSubmitted }: OrderReviewFormProps) {
  const [rating, setRating] = useState(0);
  const [comment, setComment] = useState('');
  const createReview = useCreateReview(orderId);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (rating === 0) {
      toast.error('Vui lòng chọn số sao');
      return;
    }
    try {
      await createReview.mutateAsync({
        target_type: 'STORE',
        target_id: storeId,
        rating,
        comment: comment.trim(),
      });
      toast.success('Cảm ơn bạn đã đánh giá!');
      setRating(0);
      setComment('');
      onSubmitted?.();
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Không gửi được đánh giá'));
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Đánh giá đơn hàng</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <p className="text-sm font-medium mb-2">Mức độ hài lòng</p>
            <StarRatingInput
              value={rating}
              onChange={setRating}
              disabled={createReview.isPending}
            />
          </div>
          <div>
            <label htmlFor="review-comment" className="text-sm font-medium block mb-1.5">
              Nhận xét
            </label>
            <textarea
              id="review-comment"
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              placeholder="Chia sẻ trải nghiệm của bạn về món ăn, dịch vụ…"
              rows={3}
              disabled={createReview.isPending}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50 resize-none"
            />
          </div>
          <Button
            type="submit"
            disabled={createReview.isPending || rating === 0}
            className="w-full"
          >
            {createReview.isPending ? 'Đang gửi…' : 'Gửi đánh giá'}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
