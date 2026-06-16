import type { ReactNode } from 'react';
import { ShopHeader } from '@/widgets/shop-header/shop-header';
import { ShopFooter } from '@/widgets/shop-footer/shop-footer';
import { MobileTabBar } from '@/widgets/mobile-tab-bar/mobile-tab-bar';

// Shared customer-facing chrome (header + footer + mobile tab bar) for every
// storefront/account page. Pages provide their own <main>, so this wrapper keeps
// a plain <div> to avoid nesting landmark elements. The bottom padding on mobile
// reserves room for the fixed tab bar so it never covers page content.
export function CustomerShell({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col pb-16 md:pb-0">
      <ShopHeader />
      <div className="flex-1">{children}</div>
      <ShopFooter />
      <MobileTabBar />
    </div>
  );
}
