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
import type { LocationSelection } from '@/features/locations/components/location-picker';
import { useMyLocations } from '@/features/locations/hooks/use-locations';
import { formatRoomPath } from '@/features/locations/lib/format-room-path';
import { publicPromotionApi } from '@/features/promotions/api/promotion-api';
import { formatVnd } from '@/shared/lib/format-vnd';
import { useMyWallet } from '@/features/wallet/hooks/use-wallet';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useMyCart, useCartMutations } from '@/features/cart/hooks/use-cart';
import { useStore, useShipFee } from '@/features/stores/hooks/use-stores';
import { clearActiveStoreId } from '@/shared/lib/active-store';
import { usePlaceOrder } from '../hooks/use-orders';
import type { FulfillmentType, PaymentMethod } from '../types/order';
import type { ValidatePromotionResult } from '@/features/promotions/types/promotion';

// ---- Desired-time helpers ----

// Round a minute-value up to the next multiple of 15.
function ceilToQuarter(minutes: number): number {
  return Math.ceil(minutes / 15) * 15;
}

// Parse "HH:MM" into total minutes since midnight. Returns NaN on bad input.
function parseHHMM(hhmm: string): number {
  const [h, m] = hhmm.split(':').map(Number);
  if (Number.isNaN(h) || Number.isNaN(m)) return NaN;
  return h * 60 + m;
}

// Format total minutes since midnight as "HH:MM".
function formatHHMM(totalMinutes: number): string {
  const h = Math.floor(totalMinutes / 60) % 24;
  const m = totalMinutes % 60;
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`;
}

// Build RFC3339 string for today at a given "HH:MM" time in local time.
function todayRFC3339(hhmm: string): string {
  const d = new Date();
  const [h, m] = hhmm.split(':').map(Number);
  d.setHours(h, m, 0, 0);
  return d.toISOString();
}

interface TimeSlot {
  value: string; // "HH:MM"
  label: string;
}

const EARLIEST_LABEL = 'Sớm nhất có thể';
const EARLIEST_VALUE = '__earliest__';

/**
 * Build the desired-time select options.
 *
 * Earliest slot = ceil15(now + prepMinutes).
 * Steps: every 15 minutes from earliest to CloseTimeToday.
 * First option is always "Sớm nhất có thể" (sends earliest RFC3339).
 * Returns [] when the store is closed or prep time pushes past closing.
 */
function buildTimeSlots(prepMinutes: number, closeHHMM: string): TimeSlot[] {
  const now = new Date();
  const nowMinutes = now.getHours() * 60 + now.getMinutes();
  const earliestMinutes = ceilToQuarter(nowMinutes + prepMinutes);
  const closeMinutes = parseHHMM(closeHHMM);

  if (Number.isNaN(closeMinutes) || earliestMinutes > closeMinutes) return [];

  const slots: TimeSlot[] = [];
  for (let t = earliestMinutes; t <= closeMinutes; t += 15) {
    slots.push({ value: formatHHMM(t), label: formatHHMM(t) });
  }
  return slots;
}

// ---- Component ----

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

  // Fetch store to read prep-time + open/close info.
  const { data: store } = useStore(cart?.StoreID ?? '');

  const [fulfillment, setFulfillment] = useState<FulfillmentType>('DELIVERY');
  const [locationSel, setLocationSel] = useState<LocationSelection | null>(null);
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('COD');
  const [voucherInput, setVoucherInput] = useState('');
  const [voucherResult, setVoucherResult] = useState<ValidatePromotionResult | null>(null);
  const [validatingVoucher, setValidatingVoucher] = useState(false);
  const [pickerKey, setPickerKey] = useState(0);

  // Desired-time selection: EARLIEST_VALUE = "Sớm nhất có thể" (default).
  const [desiredTimeSlot, setDesiredTimeSlot] = useState<string>(EARLIEST_VALUE);

  // Derive store open/close state and slot list.
  const prepMinutes = store?.PrepMinutes ?? 15;
  const openNow = store?.OpenNow ?? false;
  const closeHHMM = store?.CloseTimeToday ?? '';
  const openHHMM = store?.OpenTimeToday ?? '';

  const timeSlots = buildTimeSlots(prepMinutes, closeHHMM);
  // Store is orderable when it's open and there's at least one reachable slot today.
  const storeOrderable = openNow && timeSlots.length > 0;

  // Earliest reachable slot HH:MM (first item in timeSlots).
  const earliestHHMM = timeSlots[0]?.value ?? '';

  // Resolved desired_time to send: when "earliest", use the first slot RFC3339.
  function resolveDesiredTimeRFC3339(): string | undefined {
    if (!earliestHHMM) return undefined;
    const slot = desiredTimeSlot === EARLIEST_VALUE ? earliestHHMM : desiredTimeSlot;
    return todayRFC3339(slot);
  }

  // Unit ship fee for the selected delivery location.
  const shipFeeQuery = useShipFee(
    cart?.StoreID ?? '',
    fulfillment === 'DELIVERY' && locationSel ? locationSel.level : '',
    fulfillment === 'DELIVERY' && locationSel ? locationSel.id : '',
  );
  const locationNotServed =
    fulfillment === 'DELIVERY' && !!locationSel && shipFeeQuery.isError;

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
  const shipFee = fulfillment === 'DELIVERY' ? (shipFeeQuery.data?.unit_ship_fee ?? 0) : 0;
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

  const handleSavedLocationSelect = (roomId: string) => {
    setLocationSel(roomId ? { level: 'ROOM', id: roomId } : null);
    setPickerKey((k) => k + 1);
  };

  const handlePlaceOrder = async () => {
    if (!cart?.StoreID) return;
    if (fulfillment === 'DELIVERY' && !locationSel) {
      toast.error('Vui lòng chọn địa điểm giao hàng');
      return;
    }
    if (fulfillment === 'DELIVERY' && locationNotServed) {
      toast.error('Shop không giao tới đây — vui lòng chọn địa điểm khác');
      return;
    }
    if (!storeOrderable) {
      toast.error('Cửa hàng hiện không nhận đơn');
      return;
    }

    const body = {
      store_id: cart.StoreID,
      location_id: fulfillment === 'DELIVERY' ? locationSel?.id : undefined,
      location_level: fulfillment === 'DELIVERY' ? locationSel?.level : undefined,
      fulfillment,
      payment_method: paymentMethod,
      voucher_codes: voucherResult?.Applicable && voucherInput ? [voucherInput.trim().toUpperCase()] : [],
      desired_time: resolveDesiredTimeRFC3339(),
      items: items.map((it) => ({
        menu_item_id: it.MenuItemID,
        qty: it.Qty,
      })),
    };

    try {
      const result = await placeOrder.mutateAsync(body);
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
              {(savedLocs.data?.length ?? 0) > 0 && (
                <select
                  value={locationSel?.level === 'ROOM' ? locationSel.id : ''}
                  onChange={(e) => handleSavedLocationSelect(e.target.value)}
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
              <details className="text-sm">
                <summary className="text-muted-foreground cursor-pointer select-none">
                  {(savedLocs.data?.length ?? 0) > 0 ? 'Chọn địa điểm khác…' : 'Chọn địa điểm giao hàng'}
                </summary>
                <div className="mt-2">
                  <LocationPicker
                    key={pickerKey}
                    onSelectLocation={(sel) => { setLocationSel(sel); }}
                  />
                </div>
              </details>
              {locationNotServed && (
                <p className="text-destructive text-xs font-medium flex items-center gap-1">
                  Shop không giao tới đây
                </p>
              )}
            </div>
          )}

          {fulfillment === 'PICKUP' && (
            <p className="text-sm text-muted-foreground bg-muted rounded-md px-3 py-2">
              Bạn sẽ nhận mã PIN sau khi đặt hàng để tự lấy tại quán.
            </p>
          )}
        </CardContent>
      </Card>

      {/* Desired receive time */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Clock className="h-4 w-4 text-primary" />
            Giờ nhận hàng
          </CardTitle>
        </CardHeader>
        <CardContent>
          {!store ? (
            <Skeleton className="h-9 w-full" />
          ) : !storeOrderable ? (
            <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive font-medium">
              {!openNow
                ? 'Cửa hàng đang đóng cửa'
                : `Hôm nay đã hết giờ nhận đơn${closeHHMM ? ` (đóng lúc ${closeHHMM})` : ''}`}
            </div>
          ) : (
            <div className="space-y-1.5">
              <select
                value={desiredTimeSlot}
                onChange={(e) => setDesiredTimeSlot(e.target.value)}
                className="border-input bg-background focus-visible:ring-ring h-9 w-full rounded-md border px-3 text-sm focus-visible:ring-2 focus-visible:outline-none"
              >
                <option value={EARLIEST_VALUE}>{EARLIEST_LABEL} ({earliestHHMM})</option>
                {timeSlots.map((s) => (
                  <option key={s.value} value={s.value}>{s.label}</option>
                ))}
              </select>
              {openHHMM && closeHHMM && (
                <p className="text-xs text-muted-foreground">
                  Giờ hoạt động hôm nay: {openHHMM}–{closeHHMM}
                </p>
              )}
            </div>
          )}
        </CardContent>
      </Card>

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
              <span>
                {!locationSel
                  ? 'Chọn địa điểm để tính'
                  : locationNotServed
                    ? 'Không hỗ trợ'
                    : formatVnd(shipFee)}
              </span>
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
          !storeOrderable ||
          (fulfillment === 'DELIVERY' && (!locationSel || locationNotServed))
        }
      >
        {placeOrder.isPending ? 'Đang đặt hàng…' : 'Đặt hàng'}
      </Button>
    </div>
  );
}
