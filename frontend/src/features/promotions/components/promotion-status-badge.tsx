'use client';

import { Badge } from '@/shared/ui/badge';
import type { PromotionStatus } from '../types/promotion';

interface Props {
  status: PromotionStatus;
}

export function PromotionStatusBadge({ status }: Props) {
  if (status === 'ACTIVE') {
    return (
      <Badge className="bg-green-100 text-green-700 border-green-200 hover:bg-green-100">
        Đang hoạt động
      </Badge>
    );
  }
  return (
    <Badge variant="outline" className="text-muted-foreground">
      Tắt
    </Badge>
  );
}
