'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Home, Store, Search, ClipboardList, User } from 'lucide-react';
import { ROUTES } from '@/shared/config/constants';
import { cn } from '@/shared/lib/utils';
import { useAuth } from '@/features/auth/context/auth-provider';

interface Tab {
  label: string;
  href: string;
  icon: typeof Home;
  // match nested routes too (e.g. /account/orders/:id)
  match: (pathname: string) => boolean;
}

// Bottom tab bar for mobile — the familiar food-app navigation pattern
// (Home / Stores / Search / Orders / Account). Hidden on md+ where the top
// header nav takes over. The cart stays on the header bag icon (always visible).
export function MobileTabBar() {
  const pathname = usePathname();
  const { isAuthenticated } = useAuth();

  const tabs: Tab[] = [
    { label: 'Trang chủ', href: ROUTES.shop.root, icon: Home, match: (p) => p === ROUTES.shop.root },
    { label: 'Cửa hàng', href: ROUTES.stores.root, icon: Store, match: (p) => p.startsWith('/stores') },
    { label: 'Tìm món', href: ROUTES.search, icon: Search, match: (p) => p.startsWith('/search') },
    { label: 'Đơn', href: ROUTES.account.orders, icon: ClipboardList, match: (p) => p.startsWith('/account/orders') },
    {
      label: 'Tài khoản',
      href: isAuthenticated ? ROUTES.account.profile : ROUTES.auth.login,
      icon: User,
      match: (p) => (p.startsWith('/account') && !p.startsWith('/account/orders')) || p === ROUTES.auth.login,
    },
  ];

  return (
    <nav
      aria-label="Điều hướng nhanh"
      className="border-border bg-background/95 fixed inset-x-0 bottom-0 z-40 border-t pb-[env(safe-area-inset-bottom)] backdrop-blur-md md:hidden"
    >
      <ul className="mx-auto flex max-w-md items-stretch justify-around">
        {tabs.map((tab) => {
          const active = tab.match(pathname);
          const Icon = tab.icon;
          return (
            <li key={tab.href} className="flex-1">
              <Link
                href={tab.href}
                aria-current={active ? 'page' : undefined}
                className={cn(
                  'flex min-h-[3.25rem] flex-col items-center justify-center gap-0.5 px-1 py-1.5 text-[11px] font-medium transition-colors',
                  active ? 'text-primary' : 'text-muted-foreground hover:text-foreground',
                )}
              >
                <Icon className="h-5 w-5" />
                {tab.label}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
