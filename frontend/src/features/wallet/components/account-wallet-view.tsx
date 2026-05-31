'use client';

import { RoleGuard } from '@/features/auth/components/role-guard';
import { useMyWallet } from '../hooks/use-wallet';
import { WalletBalanceCard } from './wallet-balance-card';
import { WalletLedgerTable } from './wallet-ledger-table';

export function AccountWalletView() {
  return (
    <RoleGuard allow={(a) => a.isAuthenticated}>
      <main className="bg-background min-h-dvh px-4 py-10">
        <div className="mx-auto w-full max-w-2xl space-y-6">
          <h1 className="font-serif text-2xl font-bold">Ví của tôi</h1>
          <WalletContent />
        </div>
      </main>
    </RoleGuard>
  );
}

function WalletContent() {
  const { data, isLoading, isError } = useMyWallet();

  return (
    <>
      <WalletBalanceCard
        balance={data?.Balance}
        isLoading={isLoading}
      />
      <WalletLedgerTable
        entries={data?.Ledger}
        isLoading={isLoading}
        isError={isError}
      />
    </>
  );
}
