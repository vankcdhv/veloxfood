import { AdminIncidentsTable } from '@/features/deliveries/components/admin-incidents-table';
import { AdminTopbar } from '@/widgets/admin-topbar/admin-topbar';

export const metadata = {
  title: 'Sự cố giao hàng',
};

export default function AdminIncidentsPage() {
  return (
    <>
      <AdminTopbar title="Sự cố giao hàng" description="Các sự cố shipper báo cáo trong quá trình giao hàng" />
      <div className="p-4 sm:p-6 lg:p-8">
        <AdminIncidentsTable />
      </div>
    </>
  );
}
