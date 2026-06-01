import type { ReactNode } from 'react';
import { ShopHeader } from '@/widgets/shop-header/shop-header';
import { ShopFooter } from '@/widgets/shop-footer/shop-footer';

export default function SupportLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col">
      <ShopHeader />
      <main className="flex-1">{children}</main>
      <ShopFooter />
    </div>
  );
}
