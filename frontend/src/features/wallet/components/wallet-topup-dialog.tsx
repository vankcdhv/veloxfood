'use client';

import { useState } from 'react';
import { Plus, Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/shared/ui/dialog';
import { Input } from '@/shared/ui/input';
import { Label } from '@/shared/ui/label';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useTopupWallet } from '../hooks/use-wallet';

const PRESET_AMOUNTS = [50_000, 100_000, 200_000, 500_000];

export function TopupDialog() {
  const [open, setOpen] = useState(false);
  const [rawAmount, setRawAmount] = useState('');
  const topupMutation = useTopupWallet();

  const parsedAmount = parseInt(rawAmount.replace(/\D/g, ''), 10);
  const isValidAmount = !isNaN(parsedAmount) && parsedAmount >= 10_000;

  const handlePreset = (amount: number) => {
    setRawAmount(String(amount));
  };

  const handleSubmit = async () => {
    if (!isValidAmount) return;
    try {
      const result = await topupMutation.mutateAsync({ amount: parsedAmount });
      toast.success('Đang chuyển đến trang thanh toán MoMo…');
      setOpen(false);
      setRawAmount('');
      // Open MoMo sandbox pay page in a new tab.
      window.open(result.PayUrl, '_blank', 'noopener,noreferrer');
    } catch (err) {
      toast.error(getApiErrorMessage(err, 'Nạp tiền thất bại'));
    }
  };

  return (
    <>
      <Button
        size="sm"
        onClick={() => setOpen(true)}
        className="bg-orange-600 hover:bg-orange-700 text-white gap-1"
      >
        <Plus className="h-4 w-4" />
        Nạp tiền
      </Button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>Nạp tiền vào ví</DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="space-y-1.5">
              <Label htmlFor="topup-amount">Số tiền (VND)</Label>
              <Input
                id="topup-amount"
                inputMode="numeric"
                placeholder="Ví dụ: 100000"
                value={rawAmount}
                onChange={(e) => setRawAmount(e.target.value)}
              />
              {rawAmount && !isValidAmount && (
                <p className="text-destructive text-xs">Số tiền tối thiểu là 10.000đ</p>
              )}
            </div>

            <div className="flex flex-wrap gap-2">
              {PRESET_AMOUNTS.map((amt) => (
                <button
                  key={amt}
                  type="button"
                  onClick={() => handlePreset(amt)}
                  className="rounded-full border border-orange-300 px-3 py-1 text-xs font-medium text-orange-700 hover:bg-orange-50 transition-colors"
                >
                  {amt.toLocaleString('vi-VN')}đ
                </button>
              ))}
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)}>
              Huỷ
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={!isValidAmount || topupMutation.isPending}
              className="bg-orange-600 hover:bg-orange-700 text-white"
            >
              {topupMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Nạp qua MoMo
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
