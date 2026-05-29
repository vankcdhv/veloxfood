import Link from 'next/link';
import { UtensilsCrossed, Truck, Sparkles, ChefHat } from 'lucide-react';
import { ROUTES } from '@/shared/config/constants';

const HIGHLIGHTS = [
  {
    icon: UtensilsCrossed,
    title: 'Hàng ngàn món',
    desc: 'Từ quán quen tới đặc sản vùng miền',
  },
  {
    icon: Truck,
    title: 'Giao siêu tốc',
    desc: 'Trung bình 25 phút trong nội thành',
  },
  {
    icon: Sparkles,
    title: 'Ưu đãi thành viên',
    desc: 'Tích điểm mỗi đơn, đổi voucher dễ',
  },
];

export function AuthHero() {
  return (
    <aside className="from-primary relative hidden overflow-hidden bg-gradient-to-br via-orange-500 to-amber-400 lg:flex lg:flex-col lg:justify-between lg:p-10 xl:p-14">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 opacity-15"
        style={{
          backgroundImage: 'radial-gradient(circle at 1px 1px, white 1px, transparent 0)',
          backgroundSize: '20px 20px',
        }}
      />
      <div
        aria-hidden
        className="pointer-events-none absolute -top-32 -right-24 h-96 w-96 rounded-full bg-white/10 blur-3xl"
      />
      <div
        aria-hidden
        className="pointer-events-none absolute -bottom-32 -left-16 h-80 w-80 rounded-full bg-amber-200/30 blur-3xl"
      />

      <header className="relative z-10">
        <Link
          href={ROUTES.home}
          className="text-primary-foreground inline-flex items-center gap-2 transition-opacity hover:opacity-90"
        >
          <span className="inline-flex h-10 w-10 items-center justify-center rounded-lg bg-white/15 backdrop-blur-sm">
            <ChefHat className="h-5 w-5" />
          </span>
          <span className="flex flex-col leading-tight">
            <span className="font-serif text-xl font-semibold">VeloxFood</span>
            <span className="text-[10px] tracking-widest uppercase opacity-80">Food Ordering</span>
          </span>
        </Link>
      </header>

      <div className="text-primary-foreground relative z-10 space-y-6">
        <h2 className="font-serif text-4xl leading-tight font-semibold xl:text-5xl">
          Vị ngon mỗi ngày, <br /> giao tận tay bạn.
        </h2>
        <p className="max-w-md text-base text-white/85 xl:text-lg">
          Đặt món yêu thích nhanh chóng, theo dõi đơn theo thời gian thực và tích điểm cho mỗi lần
          đặt.
        </p>
      </div>

      <ul className="text-primary-foreground relative z-10 space-y-3">
        {HIGHLIGHTS.map(({ icon: Icon, title, desc }) => (
          <li
            key={title}
            className="flex items-start gap-3 rounded-lg bg-white/10 px-4 py-3 backdrop-blur-sm"
          >
            <span className="mt-0.5 inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-white/20">
              <Icon className="h-4 w-4" />
            </span>
            <span className="flex flex-col">
              <span className="text-sm font-semibold">{title}</span>
              <span className="text-xs text-white/80">{desc}</span>
            </span>
          </li>
        ))}
      </ul>
    </aside>
  );
}
