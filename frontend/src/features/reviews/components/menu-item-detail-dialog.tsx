'use client';

import { useState } from 'react';
import { UtensilsCrossed } from 'lucide-react';
import { Skeleton } from '@/shared/ui/skeleton';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog';
import type { MenuItem } from '@/features/stores/types/store';
import { useItemReviews } from '../hooks/use-reviews';
import { StarRatingDisplay } from './star-rating-input';
import { ReviewCard } from './store-reviews-list';

interface MenuItemDetailDialogProps {
  storeId: string;
  item: MenuItem;
  avg: number;
  count: number;
}

// Clickable menu item that opens a detail dialog: photo, price, full
// description, average rating and the list of reviews for that dish. The
// trigger (item name + rating line) is always shown — even with zero reviews —
// so customers can always open the detail.
export function MenuItemDetailDialog({ storeId, item, avg, count }: MenuItemDetailDialogProps) {
  const [open, setOpen] = useState(false);
  // Lazy: only fetch the item's reviews once the dialog is opened.
  const { data, isLoading, isError } = useItemReviews(storeId, item.ID, open);
  const reviews = data?.Items ?? [];

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <button
          type="button"
          className="group/detail block text-left focus-visible:outline-none"
          aria-label={`Xem chi tiết món ${item.Name}`}
        >
          <span className="font-medium text-sm leading-snug group-hover/detail:text-primary transition-colors">
            {item.Name}
          </span>
          {count > 0 ? (
            <span className="mt-0.5 flex items-center gap-1.5 text-xs text-muted-foreground">
              <StarRatingDisplay rating={Math.round(avg)} size="sm" />
              {avg.toFixed(1)} ({count})
            </span>
          ) : (
            <span className="mt-0.5 block text-xs text-muted-foreground/70">
              Chưa có đánh giá · Xem chi tiết
            </span>
          )}
        </button>
      </DialogTrigger>
      <DialogContent className="max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="text-base">{item.Name}</DialogTitle>
        </DialogHeader>

        {/* Item summary: photo + price + description + average rating */}
        <div className="flex gap-3">
          {item.ImageURL ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={item.ImageURL}
              alt={item.Name}
              className="h-20 w-20 shrink-0 rounded-lg object-cover"
            />
          ) : (
            <div className="bg-muted text-muted-foreground/40 flex h-20 w-20 shrink-0 items-center justify-center rounded-lg">
              <UtensilsCrossed className="h-7 w-7" />
            </div>
          )}
          <div className="min-w-0 flex-1 space-y-1">
            <p className="text-primary font-semibold text-sm">{item.Price.toLocaleString('vi-VN')}đ</p>
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <StarRatingDisplay rating={Math.round(avg)} size="sm" />
              {count > 0 ? `${avg.toFixed(1)} (${count} đánh giá)` : 'Chưa có đánh giá'}
            </div>
            {item.Description && (
              <p className="text-muted-foreground text-sm">{item.Description}</p>
            )}
          </div>
        </div>

        {/* Reviews for this dish */}
        <div className="border-t border-border pt-3">
          <p className="text-sm font-medium mb-2">Đánh giá món</p>
          {isLoading && (
            <div className="space-y-3">
              {Array.from({ length: 2 }).map((_, i) => (
                <Skeleton key={i} className="h-20 rounded-xl" />
              ))}
            </div>
          )}
          {isError && <p className="text-destructive text-sm py-4 text-center">Không tải được đánh giá.</p>}
          {!isLoading && !isError && reviews.length === 0 && (
            <p className="text-muted-foreground text-sm py-4 text-center">Chưa có đánh giá nào cho món này.</p>
          )}
          {reviews.length > 0 && (
            <div className="space-y-3">
              {reviews.map((review) => (
                <ReviewCard key={review.ID} review={review} />
              ))}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
