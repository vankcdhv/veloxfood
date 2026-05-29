import Link from 'next/link';
import { BrandMark } from '@/widgets/brand-mark/brand-mark';
import { ROUTES } from '@/shared/config/constants';

export function ShopFooter() {
  return (
    <footer className="border-border bg-muted/30 mt-16 border-t">
      <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
        <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-4">
          <div className="space-y-3">
            <BrandMark href={ROUTES.shop.root} />
            <p className="text-muted-foreground text-sm">
              Nền tảng đặt món ăn — frontend cho project Hệ thống phân tán.
            </p>
          </div>

          <FooterColumn
            title="Khám phá"
            links={[
              { label: 'Thực đơn', href: ROUTES.shop.menu },
              { label: 'Giỏ hàng', href: ROUTES.shop.cart },
            ]}
          />

          <FooterColumn
            title="Quản trị"
            links={[
              { label: 'Dashboard', href: ROUTES.admin.root },
              { label: 'Quản lý người dùng', href: ROUTES.admin.users },
            ]}
          />

          <FooterColumn
            title="Hỗ trợ"
            links={[
              { label: 'Trung tâm trợ giúp', href: '#' },
              { label: 'Liên hệ', href: '#' },
            ]}
          />
        </div>

        <div className="border-border text-muted-foreground mt-10 flex flex-col gap-2 border-t pt-6 text-xs sm:flex-row sm:items-center sm:justify-between">
          <p>© {new Date().getFullYear()} VeloxFood · Course project Hệ thống phân tán</p>
          <p>Made with Next.js 16 + Go microservices</p>
        </div>
      </div>
    </footer>
  );
}

interface FooterColumnProps {
  title: string;
  links: { label: string; href: string }[];
}

function FooterColumn({ title, links }: FooterColumnProps) {
  return (
    <div>
      <h3 className="text-foreground text-sm font-semibold tracking-wide uppercase">{title}</h3>
      <ul className="mt-4 space-y-2">
        {links.map((link) => (
          <li key={link.label}>
            <Link
              href={link.href}
              className="text-muted-foreground hover:text-foreground text-sm transition-colors"
            >
              {link.label}
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
