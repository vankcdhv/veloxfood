import type { Metadata } from 'next';
import { VendorStoreManagementView } from '@/features/stores/components/vendor-store-management-view';

export const metadata: Metadata = {
  title: 'Quản lý cửa hàng · VeloxFood',
};

export default function AccountStorePage() {
  return <VendorStoreManagementView />;
}
