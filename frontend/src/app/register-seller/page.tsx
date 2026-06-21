import type { Metadata } from 'next';
import { RegisterSellerView } from '@/features/sellers/components/register-seller-view';

export const metadata: Metadata = {
  title: 'Đăng ký bán hàng · VeloxFood',
};

export default function RegisterSellerPage() {
  return <RegisterSellerView />;
}
