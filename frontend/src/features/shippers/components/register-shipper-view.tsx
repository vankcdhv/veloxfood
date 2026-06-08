'use client';

import Link from 'next/link';
import { Clock, CheckCircle2, XCircle } from 'lucide-react';
import { RoleGuard } from '@/features/auth/components/role-guard';
import { Card, CardContent } from '@/shared/ui/card';
import { Button } from '@/shared/ui/button';
import { Skeleton } from '@/shared/ui/skeleton';
import { ROUTES } from '@/shared/config/constants';
import { ShipperRegistrationForm } from './shipper-registration-form';
import { useMyShipperStatus } from '../hooks/use-shippers';
import type { ShipperStatus } from '../types/shipper';

// Client wrapper so the RoleGuard `allow` predicate stays inside the client
// boundary (functions cannot be passed from a Server Component to a Client one).
export function RegisterShipperView() {
  return (
    <RoleGuard allow={(a) => a.isAuthenticated}>
      <main className="bg-background min-h-dvh px-4 py-10">
        <div className="mx-auto w-full max-w-lg">
          <h1 className="font-serif mb-6 text-center text-2xl font-bold">Trở thành Shipper</h1>
          <RegisterShipperBody />
        </div>
      </main>
    </RoleGuard>
  );
}

function RegisterShipperBody() {
  const { data, isLoading } = useMyShipperStatus();

  if (isLoading) return <Skeleton className="h-40 rounded-xl" />;
  if (data?.registered && data.status) {
    return <ShipperStatusCard status={data.status} appliedAt={data.applied_at} />;
  }
  return <ShipperRegistrationForm />;
}

function ShipperStatusCard({ status, appliedAt }: { status: ShipperStatus; appliedAt?: string }) {
  const applied = appliedAt ? new Date(appliedAt).toLocaleDateString('vi-VN') : null;

  if (status === 'approved') {
    return (
      <Card className="border-success/40 bg-success/5">
        <CardContent className="flex flex-col items-center gap-3 py-8 text-center">
          <CheckCircle2 className="text-success h-10 w-10" />
          <p className="font-semibold">Hồ sơ đã được duyệt</p>
          <p className="text-muted-foreground text-sm">Bạn có thể bắt đầu nhận đơn giao hàng.</p>
          <Button asChild className="mt-1">
            <Link href={ROUTES.account.deliveries}>Đến trang giao hàng</Link>
          </Button>
        </CardContent>
      </Card>
    );
  }

  if (status === 'rejected') {
    return (
      <Card className="border-destructive/40 bg-destructive/5">
        <CardContent className="flex flex-col items-center gap-3 py-8 text-center">
          <XCircle className="text-destructive h-10 w-10" />
          <p className="font-semibold">Hồ sơ bị từ chối</p>
          <p className="text-muted-foreground text-sm">
            Rất tiếc, hồ sơ shipper của bạn chưa được duyệt. Vui lòng liên hệ hỗ trợ để biết thêm chi tiết.
          </p>
        </CardContent>
      </Card>
    );
  }

  // pending
  return (
    <Card className="border-warning/40 bg-warning/5">
      <CardContent className="flex flex-col items-center gap-3 py-8 text-center">
        <Clock className="text-warning h-10 w-10" />
        <p className="font-semibold">Hồ sơ đang chờ duyệt</p>
        <p className="text-muted-foreground text-sm">
          Hồ sơ của bạn đang được quản trị viên xem xét{applied ? ` (gửi ngày ${applied})` : ''}. Bạn sẽ nhận được thông báo khi có kết quả.
        </p>
      </CardContent>
    </Card>
  );
}
