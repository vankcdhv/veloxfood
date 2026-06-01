'use client';

import { Wallet } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { formatVnd } from '@/shared/lib/format-vnd';
import { TopupDialog } from './wallet-topup-dialog';

interface Props {
  balance: number | undefined;
  isLoading: boolean;
}

export function WalletBalanceCard({ balance, isLoading }: Props) {
  return (
    <Card className="border-orange-200 bg-gradient-to-br from-orange-50 to-white dark:from-orange-950/20 dark:to-background">
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-base font-semibold text-orange-700 dark:text-orange-400 flex items-center gap-2">
          <Wallet className="h-5 w-5" />
          Ví VeloxFood
        </CardTitle>
        <TopupDialog />
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-9 w-40" />
        ) : (
          <p className="text-3xl font-bold text-orange-600">
            {balance !== undefined ? formatVnd(balance) : '—'}
          </p>
        )}
        <p className="text-muted-foreground mt-1 text-xs">Số dư khả dụng</p>
      </CardContent>
    </Card>
  );
}
