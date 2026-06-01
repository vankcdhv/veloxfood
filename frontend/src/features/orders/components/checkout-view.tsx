'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/shared/config/constants';
import { Tag, MapPin, Wallet, CreditCard, Smartphone, Clock } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { Skeleton } from '@/shared/ui/skeleton';
import { RoleGuard } from '@/features/auth/components/role-guard';
import { LocationPicker } from '@/features/locations/components/location-picker';
import { useMyLocations } from '@/features/locations/hooks/use-locations';
import { formatRoomPath } from '@/features/locations/lib/format-room-path';
import { publicPromotionApi } from '@/features/promotions/api/promotion-api';
import { formatVnd } from '@/shared/lib/format-vnd';
import { useMyWallet } from '@/features/wallet/hooks/use-wallet';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useMyCart, useCartMutations } from '@/features/cart/hooks/use-cart';
import { useStoreSlots, useShipFee } from '@/features/stores/hooks/use-stores';
import { isCutoffClosed, resolveActiveCutoff } from '@/features/stores/lib/slot-availability';
import { clearActiveStoreId } from '@/shared/lib/active-store';
import { usePlaceOrder } from '../hooks/use-orders';
import type { FulfillmentType, PaymentMethod } from '../types/order';
import type { ValidatePromotionResult } from '@/features/promotions/types/promotion';

// Local YYYY-MM-DD for a Date.
function isoDate(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

// Next 8 days as {value, label} for the delivery-slot date picker.
function dateOptions(): { value: string; label: string }[] {
  const base = new Date();
  return Array.from({ length: 8 }, (_, i) => {
    const d = new Date(base);
    d.setDate(base.getDate() + i);
    const label = i === 0 ? 'Hôm nay' : i === 1 ? 'Ngày mai'
      : `${String(d.getDate()).padStart(2, '0')}/${String(d.getMonth() + 1).padStart(2, '0')}`;
    return { value: isoDate(d), label };
  });
}

export function CheckoutView() {
  return (
    <RoleGuard allow={(a) => a.isAuthenticated}>
      <main className="bg-background min-h-dvh px-4 py-10">
        <div className="mx-auto w-full max-w-2xl space-y-6">
          <h1 className="font-serif text-2xl font-bold">Thanh toán</h1>
          <CheckoutContent />
        </div>
      </main>
    </RoleGuard>
  );
}

function CheckoutContent() {
  const router = useRouter();
  const { data: cart, isLoading: cartLoading } = useMyCart();
  const { clear } = useCartMutations();
  const { data: wallet } = useMyWallet();
  const savedLocs = useMyLocations();
  const placeOrder = usePlaceOrder();

  const [fulfillment, setFulfillment] = useState<FulfillmentType>('DELIVERY');
  const [locationId, setLocationId] = useState('');
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('COD');
  const [voucherInput, setVoucherInput] = useState('');
  const [voucherResult, setVoucherResult] = useState<ValidatePromotionResult | null>(null);
  const [validatingVoucher, setValidatingVoucher] = useState(false);

  // Delivery slot (ca giao) is chosen here at checkout — one slot per order — so
  // the customer always sees exactly which session they're ordering for. The menu
  // page only previews per-slot availability.
  const [dates] = useState(dateOptions);
  const [slotDate, setSlotDate] = useState(() => isoDate(new Date()));
  const [slotCutoff, setSlotCutoff] = useState('');
  const { data: slots } = useStoreSlots(cart?.StoreID ?? '', slotDate);
  const cutoffs = [...(slots?.cutoffs ?? [])].sort((a, b) => a.CutoffTime.localeCompare(b.CutoffTime));
  const hasSlots = cutoffs.length > 0;
  // The effective slot: the user's pick if still open, else the first open slot.
  // Sessions whose order deadline already passed for today are excluded.
  const activeCutoff = resolveActiveCutoff(cutoffs, slotCutoff, slotDate);
  const allSlotsClosed = hasSlots && !activeCutoff;

  // Unit ship fee for the selected delivery room (preview only).
  const { data: shipFeeData } = useShipFee(
    cart?.StoreID ?? '',
    fulfillment === 'DELIVERY' ? locationId : '',
  );

  if (cartLoading) {
    return (
      <div className="space-y-4">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-24 rounded-xl" />
        ))}
      </div>
    );
  }

  const items = cart?.Items ?? [];
  if (items.length === 0) {
    return (
      <p className="text-muted-foreground text-sm text-center py-20">
        Giỏ hàng trống. Hãy thêm món trước khi thanh toán.
      </p>
    );
  }

  const subtotal = items.reduce((s, it) => s + it.PriceSnapshot * it.Qty, 0);
  const itemDiscount = voucherResult?.Applicable ? voucherResult.ItemDiscount : 0;
  const shipDiscount = voucherResult?.Applicable ? voucherResult.ShipDiscount : 0;
  // Preview the unit ship fee for the chosen room (server stays authoritative at
  // place-order). Pickup has no ship fee.
  const shipFee = fulfillment === 'DELIVERY' ? (shipFeeData?.unit_ship_fee ?? 0) : 0;
  const grandTotal = Math.max(0, subtotal + shipFee - itemDiscount - shipDiscount);
  const walletBalance = wallet?.Balance ?? 0;
  const walletInsufficient = paymentMethod === 'WALLET' && walletBalance < grandTotal;

  const validateVoucher = async () => {
    if (!voucherInput.trim() || !cart?.StoreID) return;
    setValidatingVoucher(true);
    setVoucherResult(null);
    try {
      const result = await publicPromotionApi.validate({
        code: voucherInput.trim().toUpperCase(),
        store_id: cart.StoreID,
        subtotal,
        item_count: items.reduce((s, it) => s + it.Qty, 0),
      });
      setVoucherResult(result);
      if (result.Applicable) {
        toast.success(`Áp mã thành công — tiết kiệm ${formatVnd(result.ItemDiscount + result.ShipDiscount)}`);
      } else {
        toast.error(result.ErrorReason || 'Mã không hợp lệ');
      }
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Không kiểm tra được mã'));
    } finally {
      setValidatingVoucher(false);
    }
  };

  const handlePlaceOrder = async () => {
    if (!cart?.StoreID) return;
    if (fulfillment === 'DELIVERY' && !locationId) {
      toast.error('Vui lòng chọn địa điểm giao hàng');
      return;
    }
    if (hasSlots && !activeCutoff) {
      toast.error('Hôm nay đã hết ca giao — vui lòng chọn ngày khác.');
      return;
    }

    const body = {
      store_id: cart.StoreID,
      location_id: fulfillment === 'DELIVERY' ? locationId : undefined,
      fulfillment,
      payment_method: paymentMethod,
      voucher_codes: voucherResult?.Applicable && voucherInput ? [voucherInput.trim().toUpperCase()] : [],
      // The whole order ships in the slot chosen here; apply it to every item.
      items: items.map((it) => ({
        menu_item_id: it.MenuItemID,
        qty: it.Qty,
        cutoff_id: hasSlots ? activeCutoff : undefined,
        date: hasSlots ? slotDate : undefined,
      })),
    };

    try {
      const result = await placeOrder.mutateAsync(body);
      // Order captured the items — empty the cart so it doesn't linger.
      await clear.mutateAsync(cart.StoreID).catch(() => {});
      clearActiveStoreId();
      if (paymentMethod === 'MOMO' && result.pay_url) {
        window.location.assign(result.pay_url);
      } else {
        toast.success(`Đặt hàng thành công! Mã đơn: ${result.code}`);
        router.push(`/account/orders/${result.order_id}`);
      }
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Đặt hàng thất bại'));
    }
  };

  return (
    <div className="space-y-4">
      {/* Order items summary */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Món đã chọn</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {items.map((it) => (
            <div key={it.MenuItemID} className="flex justify-between text-sm">
              <span className="text-foreground">
                {it.NameSnapshot}
                <span className="text-muted-foreground ml-1">×{it.Qty}</span>
              </span>
              <span className="font-medium">{formatVnd(it.PriceSnapshot * it.Qty)}</span>
            </div>
          ))}
        </CardContent>
      </Card>

      {/* Fulfillment */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Hình thức nhận hàng</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex gap-2">
            {(['DELIVERY', 'PICKUP'] as FulfillmentType[]).map((f) => (
              <button
                key={f}
                type="button"
                onClick={() => setFulfillment(f)}
                className={`flex-1 rounded-lg border px-3 py-2 text-sm font-medium transition-colors ${
                  fulfillment === f
                    ? 'border-primary bg-primary/10 text-primary'
                    : 'border-border text-muted-foreground hover:border-primary/50'
                }`}
              >
                {f === 'DELIVERY' ? 'Giao tận nơi' : 'Tự lấy tại quán'}
              </button>
            ))}
          </div>

          {fulfillment === 'DELIVERY' && (
            <div className="space-y-2">
              <label className="text-sm font-medium flex items-center gap-1.5">
                <MapPin className="h-4 w-4 text-primary" />
                Địa điểm giao hàng
              </label>
              {/* Saved locations dropdown */}
              {(savedLocs.data?.length ?? 0) > 0 && (
                <select
                  value={locationId}
                  onChange={(e) => setLocationId(e.target.value)}
                  className="border-input bg-background focus-visible:ring-ring h-9 w-full rounded-md border px-3 text-sm focus-visible:ring-2 focus-visible:outline-none"
                >
                  <option value="">— Chọn vị trí đã lưu —</option>
                  {savedLocs.data?.map((loc) => (
                    <option key={loc.ID} value={loc.RoomID}>
                      {loc.Label ? `${loc.Label} – ${formatRoomPath(loc)}` : formatRoomPath(loc)}
                    </option>
                  ))}
                </select>
              )}
              {/* Cascading picker */}
              <details className="text-sm">
                <summary className="text-muted-foreground cursor-pointer select-none">
                  {(savedLocs.data?.length ?? 0) > 0 ? 'Chọn phòng khác…' : 'Chọn phòng giao hàng'}
                </summary>
                <div className="mt-2">
                  <LocationPicker
                    level="room"
                    value={locationId}
                    onSelect={setLocationId}
                  />
                </div>
              </details>
            </div>
          )}

          {fulfillment === 'PICKUP' && (
            <p className="text-sm text-muted-foreground bg-muted rounded-md px-3 py-2">
              Bạn sẽ nhận mã PIN sau khi đặt hàng để tự lấy tại quán.
            </p>
          )}
        </CardContent>
      </Card>

      {/* Delivery slot (ca giao) — only for stores that run sessions */}
      {hasSlots && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <Clock className="h-4 w-4 text-primary" />
              Ca giao
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <label className="text-foreground mb-1.5 block text-sm font-medium">Ngày</label>
              <select
                value={slotDate}
                onChange={(e) => setSlotDate(e.target.value)}
                className="border-input bg-background focus-visible:ring-ring h-9 w-full rounded-md border px-3 text-sm focus-visible:ring-2 focus-visible:outline-none"
              >
                {dates.map((d) => (
                  <option key={d.value} value={d.value}>{d.label}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-foreground mb-1.5 block text-sm font-medium">Ca</label>
              <div className="flex flex-wrap gap-2">
                {cutoffs.map((c) => {
                  const closed = isCutoffClosed(c, slotDate);
                  return (
                    <button
                      key={c.ID}
                      type="button"
                      disabled={closed}
                      title={closed ? 'Đã quá giờ cắt đơn cho ca này' : undefined}
                      onClick={() => setSlotCutoff(c.ID)}
                      className={`rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors ${
                        closed
                          ? 'border-border text-muted-foreground/40 line-through cursor-not-allowed'
                          : activeCutoff === c.ID
                            ? 'border-primary bg-primary/10 text-primary'
                            : 'border-border text-muted-foreground hover:border-primary/50'
                      }`}
                    >
                      Ca {c.CutoffTime}
                    </button>
                  );
                })}
              </div>
              {allSlotsClosed && (
                <p className="text-destructive mt-2 text-xs">
                  Hôm nay đã hết ca giao. Vui lòng chọn ngày khác ở trên.
                </p>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Voucher */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Tag className="h-4 w-4 text-primary" />
            Mã giảm giá
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <div className="flex gap-2">
            <input
              type="text"
              placeholder="Nhập mã voucher"
              value={voucherInput}
              onChange={(e) => {
                setVoucherInput(e.target.value);
                setVoucherResult(null);
              }}
              className="border-input bg-background focus-visible:ring-ring h-9 flex-1 rounded-md border px-3 text-sm uppercase focus-visible:ring-2 focus-visible:outline-none"
            />
            <Button
              variant="outline"
              size="sm"
              onClick={validateVoucher}
              disabled={!voucherInput.trim() || validatingVoucher}
            >
              {validatingVoucher ? 'Đang kiểm tra…' : 'Áp dụng'}
            </Button>
          </div>
          {voucherResult?.Applicable && (
            <div className="flex items-center gap-2 text-sm text-success">
              <Badge variant="success">Áp dụng được</Badge>
              <span>
                Giảm đơn: {formatVnd(voucherResult.ItemDiscount)} · Giảm ship: {formatVnd(voucherResult.ShipDiscount)}
              </span>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Payment method */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Phương thức thanh toán</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {(
            [
              { key: 'COD', label: 'Tiền mặt khi nhận', icon: CreditCard },
              { key: 'WALLET', label: `Ví VeloxFood (${formatVnd(walletBalance)})`, icon: Wallet },
              { key: 'MOMO', label: 'MoMo', icon: Smartphone },
            ] as { key: PaymentMethod; label: string; icon: React.ElementType }[]
          ).map(({ key, label, icon: Icon }) => {
            const walletDisabled = key === 'WALLET' && walletBalance < grandTotal;
            return (
              <button
                key={key}
                type="button"
                disabled={walletDisabled}
                onClick={() => setPaymentMethod(key)}
                title={walletDisabled ? 'Số dư ví không đủ — hãy nạp thêm' : undefined}
                className={`flex w-full items-center gap-3 rounded-lg border px-3 py-2.5 text-sm font-medium transition-colors ${
                  walletDisabled
                    ? 'border-border text-muted-foreground/50 cursor-not-allowed'
                    : paymentMethod === key
                      ? 'border-primary bg-primary/10 text-primary'
                      : 'border-border text-muted-foreground hover:border-primary/50'
                }`}
              >
                <Icon className="h-4 w-4 shrink-0" />
                {label}
                {walletDisabled && (
                  <Badge variant="destructive" className="ml-auto text-xs">Không đủ số dư</Badge>
                )}
              </button>
            );
          })}
          {walletBalance < grandTotal && (
            <p className="text-muted-foreground text-xs">
              Ví chưa đủ để thanh toán đơn này — bạn có thể{' '}
              <Link href={ROUTES.account.wallet} className="text-primary underline">nạp thêm vào ví</Link>.
            </p>
          )}
        </CardContent>
      </Card>

      {/* Summary */}
      <Card>
        <CardContent className="space-y-1.5 py-4">
          <div className="flex justify-between text-sm text-muted-foreground">
            <span>Tạm tính</span><span>{formatVnd(subtotal)}</span>
          </div>
          {fulfillment === 'DELIVERY' && (
            <div className="flex justify-between text-sm text-muted-foreground">
              <span>Phí giao hàng</span>
              <span>{locationId ? formatVnd(shipFee) : 'Chọn phòng để tính'}</span>
            </div>
          )}
          {itemDiscount > 0 && (
            <div className="flex justify-between text-sm text-success">
              <span>Giảm giá đơn</span><span>-{formatVnd(itemDiscount)}</span>
            </div>
          )}
          {shipDiscount > 0 && (
            <div className="flex justify-between text-sm text-success">
              <span>Giảm phí ship</span><span>-{formatVnd(shipDiscount)}</span>
            </div>
          )}
          <div className="flex justify-between font-semibold text-base pt-1.5 border-t border-border">
            <span>Tổng cộng</span>
            <span className="text-primary">{formatVnd(grandTotal)}</span>
          </div>
          <p className="text-xs text-muted-foreground">
            * Tổng cuối cùng được xác nhận khi đặt hàng.
          </p>
        </CardContent>
      </Card>

      <Button
        className="w-full"
        size="lg"
        onClick={handlePlaceOrder}
        disabled={
          placeOrder.isPending ||
          walletInsufficient ||
          allSlotsClosed ||
          (fulfillment === 'DELIVERY' && !locationId)
        }
      >
        {placeOrder.isPending ? 'Đang đặt hàng…' : 'Đặt hàng'}
      </Button>
    </div>
  );
}
