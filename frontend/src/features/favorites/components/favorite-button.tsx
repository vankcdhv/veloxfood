'use client';

import { Heart } from 'lucide-react';
import { toast } from 'sonner';
import { cn } from '@/shared/lib/utils';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useAuth } from '@/features/auth/context/auth-provider';
import { useFavoriteIds, useToggleFavorite } from '../hooks/use-favorites';
import type { FavoriteType } from '../api/favorite-api';

interface FavoriteButtonProps {
  type: FavoriteType;
  id: string;
  // Accessible label of the target, e.g. the store or dish name.
  name: string;
  className?: string;
}

// FavoriteButton renders the bookmark heart. Hidden when logged out (guests
// have nowhere to persist bookmarks). Optimistic: the heart flips instantly
// and rolls back on error.
export function FavoriteButton({ type, id, name, className }: FavoriteButtonProps) {
  const { user } = useAuth();
  const { idSet } = useFavoriteIds(type);
  const toggle = useToggleFavorite(type);

  if (!user) return null;

  const active = idSet.has(id);
  const onClick = (e: React.MouseEvent) => {
    // Cards are wrapped in <Link> — keep the heart from navigating.
    e.preventDefault();
    e.stopPropagation();
    toggle.mutate(
      { id, next: !active },
      {
        onError: (err) => toast.error(getApiErrorMessage(err, 'Không lưu được yêu thích')),
      },
    );
  };

  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      aria-label={active ? `Bỏ yêu thích ${name}` : `Yêu thích ${name}`}
      className={cn(
        'focus-visible:ring-ring inline-flex h-8 w-8 items-center justify-center rounded-full transition focus-visible:outline-none focus-visible:ring-2',
        active
          ? 'text-red-500 hover:text-red-600'
          : 'text-muted-foreground/50 hover:text-red-500',
        className,
      )}
    >
      <Heart className={cn('h-4.5 w-4.5', active && 'fill-current')} />
    </button>
  );
}
