import type { Metadata } from 'next';
import { ProfileView } from '@/features/auth/components/profile-view';

export const metadata: Metadata = {
  title: 'Hồ sơ của tôi · VeloxFood',
};

export default function AccountProfilePage() {
  return <ProfileView />;
}
