import type { Metadata } from 'next';
import { MyOrdersView } from '@/features/orders/components/my-orders-view';

export const metadata: Metadata = {
  title: 'Đơn hàng của tôi · VeloxFood',
};

export default function AccountOrdersPage() {
  return <MyOrdersView />;
}
