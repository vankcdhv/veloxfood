'use client';

import { useState } from 'react';
import { Star } from 'lucide-react';
import { cn } from '@/shared/lib/utils';

interface StarRatingInputProps {
  value: number;
  onChange: (rating: number) => void;
  disabled?: boolean;
}

export function StarRatingInput({ value, onChange, disabled = false }: StarRatingInputProps) {
  const [hovered, setHovered] = useState(0);

  return (
    <div className="flex gap-1" role="group" aria-label="Chọn số sao">
      {[1, 2, 3, 4, 5].map((star) => {
        const filled = star <= (hovered || value);
        return (
          <button
            key={star}
            type="button"
            disabled={disabled}
            aria-label={`${star} sao`}
            onClick={() => onChange(star)}
            onMouseEnter={() => !disabled && setHovered(star)}
            onMouseLeave={() => !disabled && setHovered(0)}
            className={cn(
              'transition-transform focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded',
              !disabled && 'hover:scale-110 cursor-pointer',
              disabled && 'cursor-default opacity-60',
            )}
          >
            <Star
              className={cn(
                'h-7 w-7 transition-colors',
                filled ? 'fill-warning text-warning' : 'text-muted-foreground',
              )}
            />
          </button>
        );
      })}
    </div>
  );
}

interface StarRatingDisplayProps {
  rating: number;
  size?: 'sm' | 'md';
}

export function StarRatingDisplay({ rating, size = 'md' }: StarRatingDisplayProps) {
  const sz = size === 'sm' ? 'h-3.5 w-3.5' : 'h-4 w-4';
  return (
    <div className="flex gap-0.5" aria-label={`${rating} sao`}>
      {[1, 2, 3, 4, 5].map((star) => (
        <Star
          key={star}
          className={cn(
            sz,
            star <= rating ? 'fill-warning text-warning' : 'text-muted-foreground/30',
          )}
        />
      ))}
    </div>
  );
}
