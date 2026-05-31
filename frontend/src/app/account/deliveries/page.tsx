import type { Metadata } from 'next';
import { ShipperDeliveryDashboard } from '@/features/deliveries/components/shipper-delivery-dashboard';

export const metadata: Metadata = {
  title: 'Giao hàng · VeloxFood',
};

export default function AccountDeliveriesPage() {
  return <ShipperDeliveryDashboard />;
}
