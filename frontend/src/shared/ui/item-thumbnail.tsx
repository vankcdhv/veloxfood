import { UtensilsCrossed } from 'lucide-react';
import { cn } from '@/shared/lib/utils';

interface ItemThumbnailProps {
  src?: string | null;
  alt: string;
  // Tailwind size classes, e.g. "h-16 w-16" (default).
  className?: string;
}

// ItemThumbnail renders a dish/store image with the standard fallback tile
// when no image exists. Shared by search results, menu lists and cards.
export function ItemThumbnail({ src, alt, className }: ItemThumbnailProps) {
  if (src) {
    return (
      // eslint-disable-next-line @next/next/no-img-element
      <img src={src} alt={alt} className={cn('h-16 w-16 shrink-0 rounded-lg object-cover', className)} />
    );
  }
  return (
    <div
      className={cn(
        'bg-muted text-muted-foreground/40 flex h-16 w-16 shrink-0 items-center justify-center rounded-lg',
        className,
      )}
      aria-hidden
    >
      <UtensilsCrossed className="h-6 w-6" />
    </div>
  );
}
