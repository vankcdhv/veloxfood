import Link from 'next/link';
import { Button } from '@/shared/ui/button';
import { BrandMark } from '@/widgets/brand-mark/brand-mark';
import { ROUTES } from '@/shared/config/constants';

export default function NotFound() {
  return (
    <main className="bg-background flex min-h-dvh flex-col items-center justify-center gap-6 px-4 text-center">
      <BrandMark href={ROUTES.shop.root} />
      <div className="space-y-2">
        <p className="text-primary font-serif text-6xl font-bold">404</p>
        <h1 className="text-xl font-semibold">Không tìm thấy trang</h1>
        <p className="text-muted-foreground text-sm">
          Trang bạn tìm không tồn tại hoặc đã được di chuyển.
        </p>
      </div>
      <Button asChild>
        <Link href={ROUTES.shop.root}>Về trang chủ</Link>
      </Button>
    </main>
  );
}
