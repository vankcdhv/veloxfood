'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { ShoppingBag, Search, Menu } from 'lucide-react';
import { Button } from '@/shared/ui/button';
import { Badge } from '@/shared/ui/badge';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/shared/ui/sheet';
import { ThemeToggle } from '@/shared/ui/theme-toggle';
import { BrandMark } from '@/widgets/brand-mark/brand-mark';
import { ROUTES } from '@/shared/config/constants';
import { cn } from '@/shared/lib/utils';
import { useAuth } from '@/features/auth/context/auth-provider';
import { UserMenu } from '@/features/auth/components/user-menu';
import { NotificationBell } from '@/features/notifications/components/notification-bell';
import { useMyCart } from '@/features/cart/hooks/use-cart';

const NAV_ITEMS = [
  { label: 'Trang chủ', href: ROUTES.shop.root },
  { label: 'Cửa hàng', href: ROUTES.stores.root },
  { label: 'Giỏ hàng', href: ROUTES.cart },
];

export function ShopHeader() {
  const pathname = usePathname();
  const { isAuthenticated, isLoading } = useAuth();
  const { data: cart } = useMyCart();
  const cartCount = cart?.Items?.reduce((s, it) => s + it.Qty, 0) ?? 0;

  return (
    <header className="border-border bg-background/80 sticky top-0 z-40 border-b backdrop-blur-md">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
        <BrandMark href={ROUTES.shop.root} />

        <nav className="hidden items-center gap-1 md:flex">
          {NAV_ITEMS.map((item) => {
            const active = pathname === item.href;
            return (
              <Link
                key={item.href}
                href={item.href}
                aria-current={active ? 'page' : undefined}
                className={cn(
                  'rounded-md px-3 py-2 text-sm font-medium transition-colors',
                  active
                    ? 'text-primary bg-primary/10'
                    : 'text-muted-foreground hover:text-foreground hover:bg-muted',
                )}
              >
                {item.label}
              </Link>
            );
          })}
        </nav>

        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon"
            aria-label="Tìm kiếm món ăn"
            className="hidden sm:inline-flex"
          >
            <Search className="h-4 w-4" />
          </Button>

          <Button variant="ghost" size="icon" aria-label="Giỏ hàng" className="relative" asChild>
            <Link href={ROUTES.cart}>
              <ShoppingBag className="h-4 w-4" />
              {cartCount > 0 && (
                <Badge
                  variant="accent"
                  className="absolute -top-1 -right-1 h-5 min-w-5 justify-center px-1 text-[10px]"
                >
                  {cartCount}
                </Badge>
              )}
            </Link>
          </Button>

          {isAuthenticated && <NotificationBell />}

          {!isLoading &&
            (isAuthenticated ? (
              <UserMenu showAdminLink />
            ) : (
              <Button variant="outline" size="sm" className="hidden sm:inline-flex" asChild>
                <Link href={ROUTES.auth.login}>Đăng nhập</Link>
              </Button>
            ))}

          <ThemeToggle />

          <Sheet>
            <SheetTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                aria-label="Mở menu điều hướng"
                className="md:hidden"
              >
                <Menu className="h-4 w-4" />
              </Button>
            </SheetTrigger>
            <SheetContent side="right" className="w-72">
              <SheetHeader>
                <SheetTitle>Điều hướng</SheetTitle>
              </SheetHeader>
              <nav className="mt-6 flex flex-col gap-1">
                {NAV_ITEMS.map((item) => (
                  <Link
                    key={item.href}
                    href={item.href}
                    className="hover:bg-muted rounded-md px-3 py-2 text-sm font-medium transition-colors"
                  >
                    {item.label}
                  </Link>
                ))}
              </nav>
            </SheetContent>
          </Sheet>
        </div>
      </div>
    </header>
  );
}
