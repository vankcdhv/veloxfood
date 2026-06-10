'use client';

import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { reviewApi } from '../api/review-api';
import { reviewKeys } from '../hooks/use-reviews';
import { StarRatingInput } from './star-rating-input';
import type { CreateReviewBody } from '../types/review';

interface OrderReviewItem {
  MenuItemID: string;
  NameSnapshot: string;
}

interface OrderReviewFormProps {
  orderId: string;
  storeId: string;
  /** Items in the order — each can be rated individually. */
  items?: OrderReviewItem[];
  /** Shipper who delivered the order — enables a shipper rating block. */
  shipperId?: string;
  /** Called after a successful submission so the parent can hide the form. */
  onSubmitted?: () => void;
}

export function OrderReviewForm({ orderId, storeId, items = [], shipperId, onSubmitted }: OrderReviewFormProps) {
  const qc = useQueryClient();
  const [storeRating, setStoreRating] = useState(0);
  const [comment, setComment] = useState('');
  // Per-item rating, keyed by menu item ID (0 = not rated → skipped).
  const [itemRatings, setItemRatings] = useState<Record<string, number>>({});
  const [shipperRating, setShipperRating] = useState(0);
  const [submitting, setSubmitting] = useState(false);

  // De-dupe items (an order may list the same dish twice with different slots).
  const uniqueItems = items.filter(
    (it, i) => items.findIndex((o) => o.MenuItemID === it.MenuItemID) === i,
  );

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (storeRating === 0) {
      toast.error('Vui lòng chấm sao cho cửa hàng');
      return;
    }

    // Build one STORE review + one ITEM review per rated dish.
    const payloads: CreateReviewBody[] = [
      { target_type: 'STORE', target_id: storeId, rating: storeRating, comment: comment.trim() },
    ];
    for (const it of uniqueItems) {
      const r = itemRatings[it.MenuItemID];
      if (r && r > 0) {
        payloads.push({ target_type: 'ITEM', target_id: it.MenuItemID, rating: r, comment: '' });
      }
    }
    if (shipperId && shipperRating > 0) {
      payloads.push({ target_type: 'SHIPPER', target_id: shipperId, rating: shipperRating, comment: '' });
    }

    setSubmitting(true);
    try {
      for (const body of payloads) {
        await reviewApi.create(orderId, body);
      }
      // Refresh everything that depends on this store's reviews.
      qc.invalidateQueries({ queryKey: reviewKeys.byStore(storeId) });
      qc.invalidateQueries({ queryKey: reviewKeys.storeSummary(storeId) });
      qc.invalidateQueries({ queryKey: reviewKeys.itemSummaries(storeId) });
      toast.success('Cảm ơn bạn đã đánh giá!');
      onSubmitted?.();
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Không gửi được đánh giá'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Đánh giá đơn hàng</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <p className="text-sm font-medium mb-2">Đánh giá cửa hàng</p>
            <StarRatingInput value={storeRating} onChange={setStoreRating} disabled={submitting} />
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
              disabled={submitting}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50 resize-none"
            />
          </div>

          {uniqueItems.length > 0 && (
            <div className="space-y-3 border-t border-border pt-4">
              <p className="text-sm font-medium">Đánh giá từng món (tuỳ chọn)</p>
              {uniqueItems.map((it) => (
                <div key={it.MenuItemID} className="flex items-center justify-between gap-3">
                  <span className="text-sm text-muted-foreground min-w-0 truncate">{it.NameSnapshot}</span>
                  <StarRatingInput
                    value={itemRatings[it.MenuItemID] ?? 0}
                    onChange={(r) => setItemRatings((prev) => ({ ...prev, [it.MenuItemID]: r }))}
                    disabled={submitting}
                  />
                </div>
              ))}
            </div>
          )}

          {shipperId && (
            <div className="space-y-2 border-t border-border pt-4">
              <p className="text-sm font-medium">Đánh giá người giao hàng (tuỳ chọn)</p>
              <StarRatingInput value={shipperRating} onChange={setShipperRating} disabled={submitting} />
            </div>
          )}

          <Button type="submit" disabled={submitting || storeRating === 0} className="w-full">
            {submitting ? 'Đang gửi…' : 'Gửi đánh giá'}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
