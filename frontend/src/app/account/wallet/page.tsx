import type { Metadata } from 'next';
import { AccountWalletView } from '@/features/wallet/components/account-wallet-view';

export const metadata: Metadata = {
  title: 'Ví của tôi · VeloxFood',
};

export default function AccountWalletPage() {
  return <AccountWalletView />;
}
