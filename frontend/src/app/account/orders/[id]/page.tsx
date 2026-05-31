import type { Metadata } from 'next';
import { OrderDetailView } from '@/features/orders/components/order-detail-view';

export const metadata: Metadata = {
  title: 'Chi tiết đơn hàng · VeloxFood',
};

interface Props {
  params: Promise<{ id: string }>;
}

export default async function OrderDetailPage({ params }: Props) {
  const { id } = await params;
  return <OrderDetailView orderId={id} />;
}
