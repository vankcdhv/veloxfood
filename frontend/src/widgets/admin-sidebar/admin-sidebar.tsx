'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { LayoutDashboard, Users, ShoppingCart, Package, Bike, MapPin, Store, Settings } from 'lucide-react';
import { BrandMark } from '@/widgets/brand-mark/brand-mark';
import { ROUTES } from '@/shared/config/constants';
import { cn } from '@/shared/lib/utils';

interface NavItem {
  label: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
}

const PRIMARY_NAV: NavItem[] = [
  { label: 'Tổng quan', href: ROUTES.admin.root, icon: LayoutDashboard },
  { label: 'Đơn hàng', href: ROUTES.admin.orders, icon: ShoppingCart },
  { label: 'Sản phẩm', href: ROUTES.admin.products, icon: Package },
  { label: 'Người dùng', href: ROUTES.admin.users, icon: Users },
  { label: 'Shipper', href: ROUTES.admin.shippers, icon: Bike },
  { label: 'Vị trí', href: ROUTES.admin.locations, icon: MapPin },
  { label: 'Cửa hàng', href: ROUTES.admin.stores, icon: Store },
];

const SECONDARY_NAV: NavItem[] = [{ label: 'Cài đặt', href: '#', icon: Settings }];

export function AdminSidebar() {
  return (
    <aside className="bg-card border-border hidden h-screen w-64 shrink-0 flex-col border-r lg:sticky lg:top-0 lg:flex">
      <div className="border-border flex h-16 items-center border-b px-6">
        <BrandMark href={ROUTES.admin.root} />
      </div>

      <nav className="flex-1 space-y-6 overflow-y-auto px-3 py-6">
        <NavSection label="Quản lý" items={PRIMARY_NAV} />
        <NavSection label="Hệ thống" items={SECONDARY_NAV} />
      </nav>

      <div className="border-border text-muted-foreground border-t p-4 text-xs">
        <p>Distributed System Course</p>
        <p>v0.1.0 · base</p>
      </div>
    </aside>
  );
}

function NavSection({ label, items }: { label: string; items: NavItem[] }) {
  const pathname = usePathname();
  return (
    <div>
      <p className="text-muted-foreground mb-2 px-3 text-[11px] font-semibold tracking-wider uppercase">
        {label}
      </p>
      <ul className="space-y-1">
        {items.map((item) => {
          const active =
            item.href === ROUTES.admin.root
              ? pathname === item.href
              : pathname.startsWith(item.href);
          const Icon = item.icon;
          return (
            <li key={item.label}>
              <Link
                href={item.href}
                aria-current={active ? 'page' : undefined}
                className={cn(
                  'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                  active
                    ? 'bg-primary/10 text-primary'
                    : 'text-muted-foreground hover:bg-muted hover:text-foreground',
                )}
              >
                <Icon className="h-4 w-4 shrink-0" />
                <span>{item.label}</span>
              </Link>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
