'use client';

import { useRef, useState } from 'react';
import { AxiosError } from 'axios';
import { useQueryClient } from '@tanstack/react-query';
import { ImagePlus, Loader2, X } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { reviewApi } from '../api/review-api';
import { reviewKeys } from '../hooks/use-reviews';
import { StarRatingInput } from './star-rating-input';
import type { CreateReviewBody } from '../types/review';

const MAX_REVIEW_PHOTOS = 3;
const MAX_PHOTO_BYTES = 5 * 1024 * 1024;

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
  // Uploaded photo URLs, attached to the STORE review on submit.
  const [photoURLs, setPhotoURLs] = useState<string[]>([]);
  const [uploading, setUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handlePickPhoto = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = ''; // allow re-selecting the same file
    if (!file) return;
    if (photoURLs.length >= MAX_REVIEW_PHOTOS) return;
    if (file.size > MAX_PHOTO_BYTES) {
      toast.error('Ảnh vượt quá 5MB.');
      return;
    }
    setUploading(true);
    try {
      const url = await reviewApi.uploadPhoto(file);
      setPhotoURLs((prev) => [...prev, url]);
    } catch (err) {
      toast.error(getApiErrorMessage(err, 'Không tải được ảnh lên'));
    } finally {
      setUploading(false);
    }
  };

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
      {
        target_type: 'STORE',
        target_id: storeId,
        rating: storeRating,
        comment: comment.trim(),
        ...(photoURLs.length > 0 ? { photo_urls: photoURLs } : {}),
      },
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
      // Already reviewed (server enforces UNIQUE order+target) — treat as success
      // so a re-submit after reload shows the thank-you state, not a scary error.
      if (e instanceof AxiosError && e.response?.status === 409) {
        toast.info('Bạn đã đánh giá đơn hàng này rồi.');
        onSubmitted?.();
      } else {
        toast.error(getApiErrorMessage(e, 'Không gửi được đánh giá'));
      }
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

          {/* Photos (attached to the store review) */}
          <div>
            <p className="text-sm font-medium mb-1.5">Ảnh món ăn (tuỳ chọn, tối đa {MAX_REVIEW_PHOTOS})</p>
            <div className="flex flex-wrap items-center gap-2">
              {photoURLs.map((url) => (
                <div key={url} className="relative">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img src={url} alt="Ảnh đánh giá đã tải lên" className="border-border h-16 w-16 rounded-lg border object-cover" />
                  <button
                    type="button"
                    onClick={() => setPhotoURLs((prev) => prev.filter((u) => u !== url))}
                    aria-label="Xoá ảnh"
                    className="bg-background border-border absolute -right-1.5 -top-1.5 rounded-full border p-0.5 shadow-sm"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </div>
              ))}
              {photoURLs.length < MAX_REVIEW_PHOTOS && (
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={uploading || submitting}
                  aria-label="Thêm ảnh đánh giá"
                  className="border-border text-muted-foreground hover:border-primary/50 hover:text-primary flex h-16 w-16 items-center justify-center rounded-lg border border-dashed transition disabled:opacity-50"
                >
                  {uploading ? <Loader2 className="h-5 w-5 animate-spin" /> : <ImagePlus className="h-5 w-5" />}
                </button>
              )}
              <input
                ref={fileInputRef}
                type="file"
                accept="image/jpeg,image/png,image/webp"
                onChange={handlePickPhoto}
                className="hidden"
              />
            </div>
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
