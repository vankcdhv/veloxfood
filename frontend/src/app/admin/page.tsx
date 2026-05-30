import { ShoppingCart, Users, DollarSign, Package } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { AdminTopbar } from '@/widgets/admin-topbar/admin-topbar';

// Số liệu tổng hợp chưa có API — hiển thị trung thực thay vì số giả gây hiểu nhầm.
const METRICS = [
  { label: 'Đơn hôm nay', value: '—', delta: 'Chưa có dữ liệu', icon: ShoppingCart, tone: 'outline' },
  { label: 'Doanh thu', value: '—', delta: 'Chưa có dữ liệu', icon: DollarSign, tone: 'outline' },
  { label: 'Người dùng mới', value: '—', delta: 'Chưa có dữ liệu', icon: Users, tone: 'outline' },
  { label: 'Sản phẩm hết', value: '—', delta: 'Chưa có dữ liệu', icon: Package, tone: 'outline' },
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
              <CardDescription>Sẽ hiển thị khi dịch vụ Đơn hàng được triển khai.</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="border-border bg-muted/40 text-muted-foreground flex h-48 items-center justify-center rounded-lg border border-dashed text-sm">
                Tính năng đang phát triển
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Dịch vụ đã triển khai</CardTitle>
              <CardDescription>Các microservice hiện có của hệ thống.</CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="space-y-3 text-sm">
                {[
                  { name: 'Tài khoản & phân quyền', svc: 'user-service' },
                  { name: 'Vị trí giao', svc: 'location-service' },
                  { name: 'Cửa hàng & thực đơn', svc: 'store-service' },
                ].map((s) => (
                  <li key={s.svc} className="flex items-center justify-between gap-2">
                    <span>{s.name}</span>
                    <Badge variant="success">Đang chạy</Badge>
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
