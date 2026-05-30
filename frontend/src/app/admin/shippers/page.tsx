import { ShipperApprovalList } from '@/features/shippers/components/shipper-approval-list';
import { AdminTopbar } from '@/widgets/admin-topbar/admin-topbar';

export const metadata = {
  title: 'Duyệt Shipper',
};

export default function AdminShippersPage() {
  return (
    <>
      <AdminTopbar title="Duyệt Shipper" description="Xem xét & duyệt hồ sơ đăng ký shipper" />
      <div className="p-4 sm:p-6 lg:p-8">
        <ShipperApprovalList />
      </div>
    </>
  );
}
