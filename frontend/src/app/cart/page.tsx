import type { Metadata } from 'next';
import { CartView } from '@/features/cart/components/cart-view';

export const metadata: Metadata = {
  title: 'Giỏ hàng · VeloxFood',
};

export default function CartPage() {
  return <CartView />;
}
