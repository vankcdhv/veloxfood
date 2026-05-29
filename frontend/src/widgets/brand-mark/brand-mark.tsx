import Link from 'next/link';
import { UtensilsCrossed } from 'lucide-react';
import { cn } from '@/shared/lib/utils';

interface BrandMarkProps {
  href?: string;
  className?: string;
}

export function BrandMark({ href = '/', className }: BrandMarkProps) {
  return (
    <Link
      href={href}
      className={cn(
        'group inline-flex items-center gap-2 transition-opacity hover:opacity-80',
        className,
      )}
    >
      <span className="bg-primary text-primary-foreground inline-flex h-9 w-9 items-center justify-center rounded-lg shadow-sm">
        <UtensilsCrossed className="h-5 w-5" />
      </span>
      <span className="flex flex-col leading-tight">
        <span className="font-serif text-lg font-semibold tracking-tight">VeloxFood</span>
        <span className="text-muted-foreground text-[10px] tracking-widest uppercase">
          Food Ordering
        </span>
      </span>
    </Link>
  );
}
