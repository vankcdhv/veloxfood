import type { Metadata } from 'next';
import { RegisterShipperView } from '@/features/shippers/components/register-shipper-view';

export const metadata: Metadata = {
  title: 'Đăng ký Shipper · VeloxFood',
};

export default function RegisterShipperPage() {
  return <RegisterShipperView />;
}
