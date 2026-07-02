import type { Metadata } from 'next';
import { FavoritesView } from '@/features/favorites/components/favorites-view';

export const metadata: Metadata = {
  title: 'Yêu thích · VeloxFood',
};

export default function AccountFavoritesPage() {
  return <FavoritesView />;
}
