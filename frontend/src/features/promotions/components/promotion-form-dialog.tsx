'use client';

import { useState } from 'react';
import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { Label } from '@/shared/ui/label';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/shared/ui/dialog';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import type { CreatePromotionBody, Promotion, PromotionType, PromotionValueKind } from '../types/promotion';
import { usePromotionMutations } from '../hooks/use-promotions';

interface Props {
  storeId: string;
  open: boolean;
  onClose: () => void;
  /** When provided the dialog is in edit mode; code is read-only. */
  promotion?: Promotion;
}

const TYPE_LABELS: Record<PromotionType, string> = {
  ORDER_DISCOUNT: 'Giảm giá đơn hàng',
  SHIP_DISCOUNT: 'Giảm phí vận chuyển',
};

const KIND_LABELS: Record<PromotionValueKind, string> = {
  PERCENT: 'Phần trăm (%)',
  AMOUNT: 'Số tiền (đ)',
};

// Convert ISO to datetime-local string (YYYY-MM-DDTHH:MM), local time.
function toInputDatetime(iso: string) {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

// Convert datetime-local input value to ISO string.
function fromInputDatetime(val: string) {
  return new Date(val).toISOString();
}

// Outer shell: owns Dialog open/close state. Remounts PromotionFormInner
// (via key) each time the dialog opens or the target promotion changes so that
// all form state is initialised fresh from props — no useEffect/setState cascade.
export function PromotionFormDialog({ storeId, open, onClose, promotion }: Props) {
  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose(); }}>
      <DialogContent className="max-w-md">
        <PromotionFormInner
          key={`${open ? 'open' : 'closed'}-${promotion?.ID ?? 'new'}`}
          storeId={storeId}
          onClose={onClose}
          promotion={promotion}
        />
      </DialogContent>
    </Dialog>
  );
}

interface InnerProps {
  storeId: string;
  onClose: () => void;
  promotion?: Promotion;
}

// Inner form: all state is initialised from props at mount time (no effect).
function PromotionFormInner({ storeId, onClose, promotion }: InnerProps) {
  const isEdit = !!promotion;
  const m = usePromotionMutations(storeId);

  const [code, setCode] = useState(promotion?.Code ?? '');
  const [type, setType] = useState<PromotionType>(promotion?.Type ?? 'ORDER_DISCOUNT');
  const [valueKind, setValueKind] = useState<PromotionValueKind>(promotion?.ValueKind ?? 'PERCENT');
  const [value, setValue] = useState(promotion ? String(promotion.Value) : '');
  const [minOrder, setMinOrder] = useState(promotion ? String(promotion.MinOrder) : '0');
  const [maxDiscount, setMaxDiscount] = useState(
    promotion?.MaxDiscount != null ? String(promotion.MaxDiscount) : '',
  );
  const [startsAt, setStartsAt] = useState(promotion ? toInputDatetime(promotion.StartsAt) : '');
  const [endsAt, setEndsAt] = useState(promotion ? toInputDatetime(promotion.EndsAt) : '');
  const [usageLimit, setUsageLimit] = useState(
    promotion?.UsageLimit != null ? String(promotion.UsageLimit) : '',
  );

  const isPending = m.create.isPending || m.update.isPending;

  const validate = (): string | null => {
    if (!isEdit && !code.trim()) return 'Vui lòng nhập mã voucher.';
    const v = parseInt(value, 10);
    if (isNaN(v) || v <= 0) return 'Giá trị giảm phải lớn hơn 0.';
    if (valueKind === 'PERCENT' && v > 100) return 'Phần trăm không được vượt quá 100.';
    const mo = parseInt(minOrder, 10);
    if (isNaN(mo) || mo < 0) return 'Đơn tối thiểu không hợp lệ.';
    if (!startsAt || !endsAt) return 'Vui lòng chọn thời gian áp dụng.';
    if (new Date(startsAt) >= new Date(endsAt)) return 'Thời gian kết thúc phải sau thời gian bắt đầu.';
    return null;
  };

  const handleSubmit = () => {
    const err = validate();
    if (err) { toast.error(err); return; }

    const baseFields = {
      value: parseInt(value, 10),
      min_order: parseInt(minOrder, 10),
      max_discount: maxDiscount.trim() ? parseInt(maxDiscount, 10) : undefined,
      starts_at: fromInputDatetime(startsAt),
      ends_at: fromInputDatetime(endsAt),
      usage_limit: usageLimit.trim() ? parseInt(usageLimit, 10) : undefined,
    };

    if (isEdit && promotion) {
      m.update.mutate(
        { promoId: promotion.ID, body: baseFields },
        {
          onSuccess: () => { toast.success('Đã cập nhật voucher.'); onClose(); },
          onError: (e) => toast.error(getApiErrorMessage(e, 'Cập nhật thất bại')),
        },
      );
    } else {
      const body: CreatePromotionBody = {
        code: code.trim().toUpperCase(),
        type: type,
        value_kind: valueKind,
        ...baseFields,
      };
      m.create.mutate(body, {
        onSuccess: () => { toast.success('Đã tạo voucher.'); onClose(); },
        onError: (e) => toast.error(getApiErrorMessage(e, 'Tạo voucher thất bại')),
      });
    }
  };

  return (
    <>
      <DialogHeader>
        <DialogTitle>{isEdit ? 'Chỉnh sửa voucher' : 'Tạo voucher mới'}</DialogTitle>
      </DialogHeader>

      <div className="space-y-4 py-2">
        {/* Code — read-only in edit mode */}
        <div className="space-y-1">
          <Label htmlFor="promo-code">Mã voucher</Label>
          <Input
            id="promo-code"
            value={code}
            onChange={(e) => setCode(e.target.value.toUpperCase())}
            disabled={isEdit}
            placeholder="VD: SALE20"
            className={isEdit ? 'opacity-60' : ''}
          />
          {isEdit && <p className="text-muted-foreground text-xs">Mã không thể thay đổi sau khi tạo.</p>}
        </div>

        {/* Type + ValueKind (create only) */}
        {!isEdit && (
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label htmlFor="promo-type">Loại giảm</Label>
              <select
                id="promo-type"
                value={type}
                onChange={(e) => setType(e.target.value as PromotionType)}
                className="border-input bg-background h-9 w-full rounded-md border px-2 text-sm"
              >
                {(Object.keys(TYPE_LABELS) as PromotionType[]).map((k) => (
                  <option key={k} value={k}>{TYPE_LABELS[k]}</option>
                ))}
              </select>
            </div>
            <div className="space-y-1">
              <Label htmlFor="promo-kind">Kiểu giá trị</Label>
              <select
                id="promo-kind"
                value={valueKind}
                onChange={(e) => setValueKind(e.target.value as PromotionValueKind)}
                className="border-input bg-background h-9 w-full rounded-md border px-2 text-sm"
              >
                {(Object.keys(KIND_LABELS) as PromotionValueKind[]).map((k) => (
                  <option key={k} value={k}>{KIND_LABELS[k]}</option>
                ))}
              </select>
            </div>
          </div>
        )}

        {/* Value */}
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label htmlFor="promo-value">
              Giá trị giảm {!isEdit && valueKind === 'PERCENT' ? '(%)' : '(đ)'}
            </Label>
            <Input
              id="promo-value"
              type="number"
              min={1}
              max={!isEdit && valueKind === 'PERCENT' ? 100 : undefined}
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder={!isEdit && valueKind === 'PERCENT' ? '10' : '5000'}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="promo-min-order">Đơn tối thiểu (đ)</Label>
            <Input
              id="promo-min-order"
              type="number"
              min={0}
              value={minOrder}
              onChange={(e) => setMinOrder(e.target.value)}
              placeholder="0"
            />
          </div>
        </div>

        {/* MaxDiscount + UsageLimit */}
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label htmlFor="promo-max">Giảm tối đa (đ, tuỳ chọn)</Label>
            <Input
              id="promo-max"
              type="number"
              min={0}
              value={maxDiscount}
              onChange={(e) => setMaxDiscount(e.target.value)}
              placeholder="Không giới hạn"
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="promo-usage-limit">Giới hạn lượt (tuỳ chọn)</Label>
            <Input
              id="promo-usage-limit"
              type="number"
              min={1}
              value={usageLimit}
              onChange={(e) => setUsageLimit(e.target.value)}
              placeholder="Không giới hạn"
            />
          </div>
        </div>

        {/* Date range */}
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label htmlFor="promo-starts">Bắt đầu</Label>
            <Input
              id="promo-starts"
              type="datetime-local"
              value={startsAt}
              onChange={(e) => setStartsAt(e.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="promo-ends">Kết thúc</Label>
            <Input
              id="promo-ends"
              type="datetime-local"
              value={endsAt}
              onChange={(e) => setEndsAt(e.target.value)}
            />
          </div>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" onClick={onClose} disabled={isPending}>
          Huỷ
        </Button>
        <Button
          onClick={handleSubmit}
          disabled={isPending}
          className="bg-orange-600 hover:bg-orange-700 text-white"
        >
          {isPending
            ? <Loader2 className="h-4 w-4 animate-spin" />
            : isEdit ? 'Lưu thay đổi' : 'Tạo voucher'}
        </Button>
      </DialogFooter>
    </>
  );
}
