import type { Metadata } from 'next';
import { AccountLocationsView } from '@/features/locations/components/account-locations-view';

export const metadata: Metadata = {
  title: 'Vị trí giao của tôi · VeloxFood',
};

export default function AccountLocationsPage() {
  return <AccountLocationsView />;
}
