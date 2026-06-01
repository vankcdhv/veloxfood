import type { ReactNode } from 'react';
import { AdminSidebar, AdminMobileTopbar } from '@/widgets/admin-sidebar/admin-sidebar';
import { AdminGuard } from '@/features/auth/components/admin-guard';

export default function AdminLayout({ children }: { children: ReactNode }) {
  return (
    <AdminGuard>
      <div className="bg-muted/30 flex min-h-screen">
        <AdminSidebar />
        <div className="flex min-w-0 flex-1 flex-col">
          <AdminMobileTopbar />
          {children}
        </div>
      </div>
    </AdminGuard>
  );
}
