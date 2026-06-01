import type { ReactNode } from 'react';
import { ShopHeader } from '@/widgets/shop-header/shop-header';
import { ShopFooter } from '@/widgets/shop-footer/shop-footer';

// Shared customer-facing chrome (header + footer) for every storefront/account
// page. Pages provide their own <main>, so this wrapper keeps a plain <div> to
// avoid nesting landmark elements.
export function CustomerShell({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col">
      <ShopHeader />
      <div className="flex-1">{children}</div>
      <ShopFooter />
    </div>
  );
}
