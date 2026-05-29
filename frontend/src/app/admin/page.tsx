import { ShoppingCart, Users, DollarSign, Package } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { AdminTopbar } from '@/widgets/admin-topbar/admin-topbar';

const METRICS = [
  { label: 'Đơn hôm nay', value: '128', delta: '+12%', icon: ShoppingCart, tone: 'success' },
  { label: 'Doanh thu', value: '24.8M', delta: '+8.4%', icon: DollarSign, tone: 'accent' },
  { label: 'Người dùng mới', value: '42', delta: '+5', icon: Users, tone: 'warning' },
  { label: 'Sản phẩm hết', value: '3', delta: 'cần nhập', icon: Package, tone: 'destructive' },
] as const;

export default function AdminDashboardPage() {
  return (
    <>
      <AdminTopbar title="Tổng quan" description="Snapshot vận hành 24h qua" />

      <div className="space-y-6 p-4 sm:p-6 lg:p-8">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {METRICS.map((m) => {
            const Icon = m.icon;
            return (
              <Card key={m.label}>
                <CardContent className="p-5">
                  <div className="flex items-start justify-between">
                    <div>
                      <p className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                        {m.label}
                      </p>
                      <p className="mt-2 font-serif text-3xl font-bold">{m.value}</p>
                    </div>
                    <span className="bg-primary/10 text-primary inline-flex h-9 w-9 items-center justify-center rounded-lg">
                      <Icon className="h-4 w-4" />
                    </span>
                  </div>
                  <Badge variant={m.tone} className="mt-3">
                    {m.delta}
                  </Badge>
                </CardContent>
              </Card>
            );
          })}
        </div>

        <div className="grid gap-6 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Đơn hàng gần đây</CardTitle>
              <CardDescription>10 đơn hàng mới nhất sẽ hiển thị tại đây.</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="border-border bg-muted/40 text-muted-foreground flex h-48 items-center justify-center rounded-lg border border-dashed text-sm">
                [Bảng đơn hàng — placeholder, dựng ở phase sau]
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Hoạt động hệ thống</CardTitle>
              <CardDescription>Trạng thái microservices.</CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="space-y-3 text-sm">
                {[
                  { name: 'user-service', status: 'healthy', tone: 'success' as const },
                  { name: 'order-service', status: 'pending', tone: 'warning' as const },
                  { name: 'payment-service', status: 'down', tone: 'destructive' as const },
                ].map((s) => (
                  <li key={s.name} className="flex items-center justify-between">
                    <span className="font-mono">{s.name}</span>
                    <Badge variant={s.tone}>{s.status}</Badge>
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>
        </div>
      </div>
    </>
  );
}
