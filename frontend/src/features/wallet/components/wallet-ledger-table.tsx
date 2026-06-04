'use client';

import { ArrowDownLeft, ArrowUpRight, RefreshCw, TrendingDown } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { formatVnd } from '@/shared/lib/format-vnd';
import type { EntryType, LedgerEntry } from '../types/wallet';

interface Props {
  entries: LedgerEntry[] | undefined;
  isLoading: boolean;
  isError: boolean;
}

const ENTRY_LABELS: Record<EntryType, string> = {
  TOPUP: 'Nạp tiền',
  PAYMENT: 'Thanh toán',
  REFUND: 'Hoàn tiền',
  PAYOUT: 'Chi trả',
};

const ENTRY_ICONS: Record<EntryType, React.ReactNode> = {
  TOPUP: <ArrowDownLeft className="h-4 w-4 text-green-600" />,
  PAYMENT: <ArrowUpRight className="h-4 w-4 text-orange-600" />,
  REFUND: <RefreshCw className="h-4 w-4 text-blue-600" />,
  PAYOUT: <TrendingDown className="h-4 w-4 text-red-600" />,
};

function isCredit(type: EntryType): boolean {
  return type === 'TOPUP' || type === 'REFUND';
}

function formatDatetime(iso: string): string {
  return new Date(iso).toLocaleString('vi-VN', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function WalletLedgerTable({ entries, isLoading, isError }: Props) {
  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base">Lịch sử giao dịch</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading && (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-12 rounded-lg" />
            ))}
          </div>
        )}

        {isError && (
          <p className="text-destructive text-sm py-4 text-center">
            Không tải được lịch sử giao dịch.
          </p>
        )}

        {!isLoading && !isError && (!entries || entries.length === 0) && (
          <p className="text-muted-foreground text-sm py-8 text-center">
            Chưa có giao dịch nào.
          </p>
        )}

        {!isLoading && !isError && entries && entries.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-muted-foreground text-left">
                  <th className="pb-2 pr-3 font-medium">Loại</th>
                  <th className="pb-2 pr-3 font-medium text-right">Số tiền</th>
                  <th className="pb-2 pr-3 font-medium text-right">Số dư sau</th>
                  <th className="pb-2 font-medium whitespace-nowrap">Thời gian</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {entries.map((entry) => (
                  <LedgerRow key={entry.ID} entry={entry} />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function LedgerRow({ entry }: { entry: LedgerEntry }) {
  const credit = isCredit(entry.EntryType);
  const label = ENTRY_LABELS[entry.EntryType] ?? 'Giao dịch khác';
  const icon = ENTRY_ICONS[entry.EntryType] ?? <RefreshCw className="h-4 w-4 text-muted-foreground" />;

  return (
    <tr className="hover:bg-muted/40 transition-colors">
      <td className="py-3 pr-3">
        <div className="flex items-center gap-2">
          {icon}
          <span className="font-medium">{label}</span>
        </div>
      </td>
      <td className={`py-3 pr-3 text-right font-semibold tabular-nums ${credit ? 'text-green-600' : 'text-orange-600'}`}>
        {credit ? '+' : '−'}{formatVnd(Math.abs(entry.Amount))}
      </td>
      <td className="py-3 pr-3 text-right text-muted-foreground tabular-nums">
        {formatVnd(entry.BalanceAfter)}
      </td>
      <td className="py-3 text-muted-foreground whitespace-nowrap">
        {formatDatetime(entry.CreatedAt)}
      </td>
    </tr>
  );
}
