'use client';

import Link from 'next/link';
import { Button } from '@/shared/ui/button';
import { BrandMark } from '@/widgets/brand-mark/brand-mark';
import { ROUTES } from '@/shared/config/constants';

export default function AppError({ reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <main className="bg-background flex min-h-dvh flex-col items-center justify-center gap-6 px-4 text-center">
      <BrandMark href={ROUTES.shop.root} />
      <div className="space-y-2">
        <h1 className="text-xl font-semibold">Đã có lỗi xảy ra</h1>
        <p className="text-muted-foreground text-sm">
          Xin lỗi, đã có sự cố không mong muốn. Vui lòng thử lại.
        </p>
      </div>
      <div className="flex gap-3">
        <Button onClick={reset}>Thử lại</Button>
        <Button asChild variant="outline">
          <Link href={ROUTES.shop.root}>Về trang chủ</Link>
        </Button>
      </div>
    </main>
  );
}
