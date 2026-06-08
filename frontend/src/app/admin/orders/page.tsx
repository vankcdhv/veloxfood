import { AdminTopbar } from '@/widgets/admin-topbar/admin-topbar';
import { AdminOrdersTable } from '@/features/orders/components/admin-orders-table';

export const metadata = {
  title: 'Đơn hàng',
};

export default function AdminOrdersPage() {
  return (
    <>
      <AdminTopbar title="Đơn hàng" description="Giám sát đơn hàng gần đây trên toàn hệ thống" />
      <div className="p-4 sm:p-6 lg:p-8">
        <AdminOrdersTable />
      </div>
    </>
  );
}
