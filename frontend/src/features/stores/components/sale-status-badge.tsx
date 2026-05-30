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
