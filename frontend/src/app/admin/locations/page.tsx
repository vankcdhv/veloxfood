import { LocationTreeManager } from '@/features/locations/components/location-tree-manager';
import { AdminTopbar } from '@/widgets/admin-topbar/admin-topbar';

export const metadata = {
  title: 'Vị trí giao',
};

export default function AdminLocationsPage() {
  return (
    <>
      <AdminTopbar title="Cây vị trí giao" description="Quản lý Toà nhà → Tầng → Phòng" />
      <div className="p-4 sm:p-6 lg:p-8">
        <LocationTreeManager />
      </div>
    </>
  );
}
