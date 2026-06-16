'use client';

import Link from 'next/link';
import { Clock, Package, Truck, UtensilsCrossed } from 'lucide-react';
import { Card, CardContent } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { StarRatingDisplay } from '@/features/reviews/components/star-rating-input';
import { useStoreRatingSummary } from '@/features/reviews/hooks/use-reviews';
import { StoreOpenBadge } from './sale-status-badge';
import type { Store } from '../types/store';

interface StoreCardProps {
  store: Store;
  href?: string;
}

export function StoreCard({ store, href }: StoreCardProps) {
  const { data: rating } = useStoreRatingSummary(store.ID);

  // h-full + flex column so every card fills its grid cell and lines up evenly;
  // the footer is pinned to the bottom (mt-auto) regardless of body length.
  const content = (
    <Card className="group h-full flex flex-col cursor-pointer transition hover:shadow-md hover:-translate-y-0.5">
      <CardContent className="flex flex-1 flex-col p-5">
        {/* Header: cuisine icon + name (clamped) + status badge (no wrap) */}
        <div className="flex items-start gap-3">
          {store.AvatarURL ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={store.AvatarURL} alt={store.Name} className="h-10 w-10 shrink-0 rounded-lg object-cover" />
          ) : (
            <div className="bg-primary/10 text-primary flex h-10 w-10 shrink-0 items-center justify-center rounded-lg">
              <UtensilsCrossed className="h-5 w-5" />
            </div>
          )}
          <div className="min-w-0 flex-1">
            <h3 className="font-serif text-base font-semibold leading-snug line-clamp-2 group-hover:text-primary transition-colors">
              {store.Name}
            </h3>
            {store.BusinessType && (
              <p className="text-muted-foreground text-xs line-clamp-1 mt-0.5">{store.BusinessType}</p>
            )}
          </div>
          <StoreOpenBadge openNow={store.OpenNow} status={store.SaleStatus} />
        </div>

        {/* Rating row — always reserved so cards stay uniform */}
        <div className="mt-3 flex items-center gap-1.5 text-xs">
          {rating && rating.Count > 0 ? (
            <>
              <StarRatingDisplay rating={Math.round(rating.Avg)} size="sm" />
              <span className="font-medium">{rating.Avg.toFixed(1)}</span>
              <span className="text-muted-foreground">({rating.Count})</span>
            </>
          ) : (
            <span className="text-muted-foreground/60">Chưa có đánh giá</span>
          )}
        </div>

        {/* Meta row: prep-time ETA + today's hours (data already returned by the
            browse endpoint) — gives users decision info like ShopeeFood/GrabFood. */}
        <div className="text-muted-foreground mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
          <span className="inline-flex items-center gap-1">
            <Clock className="h-3 w-3" />~{store.PrepMinutes && store.PrepMinutes > 0 ? store.PrepMinutes : 20} phút
          </span>
          {store.OpenTimeToday && store.CloseTimeToday && (
            <span>
              Mở {store.OpenTimeToday}–{store.CloseTimeToday}
            </span>
          )}
        </div>

        {/* Footer chips pinned to the bottom */}
        <div className="mt-auto flex flex-wrap items-center gap-2 pt-4">
          <Badge variant="outline" className="text-xs gap-1">
            <Truck className="h-3 w-3" />
            Giao tận nơi
          </Badge>
          {store.PickupEnabled && (
            <Badge variant="secondary" className="text-xs gap-1">
              <Package className="h-3 w-3" />
              Tự đến lấy
            </Badge>
          )}
        </div>
      </CardContent>
    </Card>
  );

  if (href) {
    return (
      <Link href={href} className="block h-full">
        {content}
      </Link>
    );
  }
  return content;
}
