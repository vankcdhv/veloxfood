import type { Metadata } from 'next';
import { CheckoutView } from '@/features/orders/components/checkout-view';

export const metadata: Metadata = {
  title: 'Thanh toán · VeloxFood',
};

export default function CheckoutPage() {
  return <CheckoutView />;
}
