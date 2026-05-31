'use client';

import { Badge } from '@/shared/ui/badge';
import type { DeliveryStatus } from '../types/delivery';

type BadgeVariant = 'default' | 'warning' | 'success' | 'destructive' | 'secondary' | 'accent' | 'outline';

const STATUS_MAP: Record<DeliveryStatus, { label: string; variant: BadgeVariant }> = {
  AVAILABLE:        { label: 'Chờ nhận',       variant: 'warning' },
  CLAIMED:          { label: 'Đã nhận',         variant: 'accent' },
  PICKED_UP:        { label: 'Đã lấy hàng',     variant: 'accent' },
  DELIVERING:       { label: 'Đang giao',        variant: 'accent' },
  DELIVERED:        { label: 'Đã giao',          variant: 'success' },
  CANCELLED:        { label: 'Đã hủy',           variant: 'destructive' },
  STORE_DELIVERING: { label: 'Cửa hàng tự giao', variant: 'secondary' },
};

export function DeliveryStatusBadge({ status }: { status: DeliveryStatus }) {
  const { label, variant } = STATUS_MAP[status] ?? { label: status, variant: 'outline' as BadgeVariant };
  return <Badge variant={variant}>{label}</Badge>;
}

export function deliveryStatusLabel(status: DeliveryStatus): string {
  return STATUS_MAP[status]?.label ?? status;
}
