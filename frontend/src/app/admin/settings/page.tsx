import { AdminTopbar } from '@/widgets/admin-topbar/admin-topbar';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';

export const metadata = { title: 'Cài đặt hệ thống' };

const PLATFORM_INFO = [
  { label: 'Tên hệ thống', value: 'VeloxFood' },
  { label: 'Phiên bản', value: '1.0' },
];

const PAYMENT_INFO = [
  { label: 'COD (tiền mặt)', value: 'Bật' },
  { label: 'Ví VeloxFood', value: 'Bật' },
  { label: 'MoMo', value: 'Bật (sandbox)' },
];

export default function AdminSettingsPage() {
  return (
    <>
      <AdminTopbar title="Cài đặt hệ thống" description="Thông tin nền tảng và cấu hình thanh toán" />
      <div className="grid gap-6 p-4 sm:p-6 lg:grid-cols-2 lg:p-8">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Thông tin nền tảng</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {PLATFORM_INFO.map((r) => (
              <div key={r.label} className="flex justify-between text-sm">
                <span className="text-muted-foreground">{r.label}</span>
                <span className="font-medium">{r.value}</span>
              </div>
            ))}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Phương thức thanh toán</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {PAYMENT_INFO.map((r) => (
              <div key={r.label} className="flex justify-between text-sm">
                <span className="text-muted-foreground">{r.label}</span>
                <span className="font-medium">{r.value}</span>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
    </>
  );
}
