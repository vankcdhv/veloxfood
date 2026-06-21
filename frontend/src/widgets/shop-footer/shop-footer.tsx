'use client';

import Link from 'next/link';
import { BrandMark } from '@/widgets/brand-mark/brand-mark';
import { ROUTES } from '@/shared/config/constants';
import { useAuth } from '@/features/auth/context/auth-provider';

export function ShopFooter() {
  const { isAdmin } = useAuth();
  return (
    <footer className="border-border bg-muted/30 mt-16 border-t">
      <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
        <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-4">
          <div className="space-y-3">
            <BrandMark href={ROUTES.shop.root} />
            <p className="text-muted-foreground text-sm">
              Món ngon mỗi ngày, giao tận tay bạn.
            </p>
          </div>

          <FooterColumn
            title="Khám phá"
            links={[
              { label: 'Cửa hàng', href: ROUTES.stores.root },
              { label: 'Giỏ hàng', href: ROUTES.cart },
            ]}
          />

          {isAdmin && (
            <FooterColumn
              title="Quản trị"
              links={[
                { label: 'Dashboard', href: ROUTES.admin.root },
                { label: 'Quản lý người dùng', href: ROUTES.admin.users },
              ]}
            />
          )}

          <FooterColumn
            title="Hỗ trợ"
            links={[
              { label: 'Trung tâm trợ giúp', href: ROUTES.support.help },
              { label: 'Liên hệ', href: ROUTES.support.contact },
            ]}
          />
        </div>

        <div className="border-border text-muted-foreground mt-10 flex flex-col gap-2 border-t pt-6 text-xs sm:flex-row sm:items-center sm:justify-between">
          <p>© {new Date().getFullYear()} VeloxFood</p>
          <p>Đặt món ngon — nhanh, dễ, đúng vị.</p>
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
