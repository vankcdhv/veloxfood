'use client';

import { ChevronLeft, ChevronRight } from 'lucide-react';
import { Button } from '@/shared/ui/button';

interface PaginationProps {
  page: number;
  totalPages: number;
  onChange: (page: number) => void;
  className?: string;
}

// Numbered prev/next pagination control for data tables (admin / owner views).
// Hidden when there is only a single page.
export function Pagination({ page, totalPages, onChange, className }: PaginationProps) {
  if (totalPages <= 1) return null;
  return (
    <div className={`flex items-center justify-center gap-4 pt-2 ${className ?? ''}`}>
      <Button
        variant="outline"
        size="sm"
        className="gap-1"
        disabled={page <= 1}
        onClick={() => onChange(Math.max(1, page - 1))}
      >
        <ChevronLeft className="h-4 w-4" />
        Trước
      </Button>
      <span className="text-muted-foreground text-sm">
        Trang {page}/{totalPages}
      </span>
      <Button
        variant="outline"
        size="sm"
        className="gap-1"
        disabled={page >= totalPages}
        onClick={() => onChange(Math.min(totalPages, page + 1))}
      >
        Sau
        <ChevronRight className="h-4 w-4" />
      </Button>
    </div>
  );
}
