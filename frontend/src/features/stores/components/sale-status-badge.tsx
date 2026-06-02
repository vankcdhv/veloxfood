import { Badge } from '@/shared/ui/badge';
import type { SaleStatus } from '../types/store';

const CONFIG: Record<SaleStatus, { variant: 'success' | 'destructive' | 'warning'; label: string }> = {
  OPEN: { variant: 'success', label: 'Đang mở' },
  CLOSED_TODAY: { variant: 'destructive', label: 'Đóng hôm nay' },
  PAUSED: { variant: 'warning', label: 'Tạm dừng' },
};

export function SaleStatusBadge({ status }: { status: SaleStatus }) {
  const cfg = CONFIG[status] ?? { variant: 'outline' as const, label: status };
  return <Badge variant={cfg.variant}>{cfg.label}</Badge>;
}

// Customer-facing badge that reflects whether the store is actually open *right now*
// (SaleStatus + today's operating hours), matching the checkout gate. A store that
// is OPEN but outside its hours (or has none today) shows "Ngoài giờ" — not "Đang mở".
export function StoreOpenBadge({ openNow, status }: { openNow?: boolean; status: SaleStatus }) {
  if (openNow) return <Badge variant="success">Đang mở</Badge>;
  if (status === 'PAUSED') return <Badge variant="warning">Tạm dừng</Badge>;
  if (status === 'CLOSED_TODAY') return <Badge variant="destructive">Đóng hôm nay</Badge>;
  return <Badge variant="outline">Ngoài giờ</Badge>;
}
