import type { ReactNode } from 'react';
import { CustomerShell } from '@/widgets/customer-shell/customer-shell';

export default function CartLayout({ children }: { children: ReactNode }) {
  return <CustomerShell>{children}</CustomerShell>;
}
