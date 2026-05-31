'use client';

import { Badge } from '@/shared/ui/badge';
import type { OrderStatus } from '../types/order';

const STATUS_MAP: Record<OrderStatus, { label: string; variant: 'default' | 'warning' | 'success' | 'destructive' | 'secondary' | 'accent' | 'outline' }> = {
  PENDING:    { label: 'Chờ xác nhận', variant: 'warning' },
  CONFIRMED:  { label: 'Đã xác nhận',  variant: 'accent' },
  PREPARING:  { label: 'Đang chuẩn bị', variant: 'accent' },
  READY:      { label: 'Sẵn sàng',      variant: 'success' },
  DELIVERING: { label: 'Đang giao',     variant: 'accent' },
  DELIVERED:  { label: 'Đã giao',       variant: 'success' },
  COMPLETED:  { label: 'Hoàn thành',    variant: 'success' },
  CANCELLED:  { label: 'Đã hủy',        variant: 'destructive' },
};

export function OrderStatusBadge({ status }: { status: OrderStatus }) {
  const { label, variant } = STATUS_MAP[status] ?? { label: status, variant: 'outline' };
  return <Badge variant={variant}>{label}</Badge>;
}

export function orderStatusLabel(status: OrderStatus): string {
  return STATUS_MAP[status]?.label ?? status;
}
