'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Tag, MapPin, Wallet, CreditCard, Smartphone } from 'lucide-react';
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
import { formatVnd } from '@/features/wallet/lib/format-vnd';
import { useMyWallet } from '@/features/wallet/hooks/use-wallet';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useMyCart } from '@/features/cart/hooks/use-cart';
import { usePlaceOrder } from '../hooks/use-orders';
import type { FulfillmentType, PaymentMethod } from '../types/order';
import type { ValidatePromotionResult } from '@/features/promotions/types/promotion';

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
  const { data: wallet } = useMyWallet();
  const savedLocs = useMyLocations();
  const placeOrder = usePlaceOrder();

  const [fulfillment, setFulfillment] = useState<FulfillmentType>('DELIVERY');
  const [locationId, setLocationId] = useState('');
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('COD');
  const [voucherInput, setVoucherInput] = useState('');
  const [voucherResult, setVoucherResult] = useState<ValidatePromotionResult | null>(null);
  const [validatingVoucher, setValidatingVoucher] = useState(false);

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
  // Ship fee is resolved server-side at place-order; preview 0 here.
  const grandTotal = Math.max(0, subtotal - itemDiscount - shipDiscount);
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

    const body = {
      store_id: cart.StoreID,
      location_id: fulfillment === 'DELIVERY' ? locationId : undefined,
      fulfillment,
      payment_method: paymentMethod,
      voucher_codes: voucherResult?.Applicable && voucherInput ? [voucherInput.trim().toUpperCase()] : [],
      items: items.map((it) => ({
        menu_item_id: it.MenuItemID,
        qty: it.Qty,
        cutoff_id: it.CutoffID,
        date: it.Date,
      })),
    };

    try {
      const result = await placeOrder.mutateAsync(body);
      if (paymentMethod === 'MOMO' && result.pay_url) {
        window.location.href = result.pay_url;
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
          ).map(({ key, label, icon: Icon }) => (
            <button
              key={key}
              type="button"
              onClick={() => setPaymentMethod(key)}
              className={`flex w-full items-center gap-3 rounded-lg border px-3 py-2.5 text-sm font-medium transition-colors ${
                paymentMethod === key
                  ? 'border-primary bg-primary/10 text-primary'
                  : 'border-border text-muted-foreground hover:border-primary/50'
              }`}
            >
              <Icon className="h-4 w-4 shrink-0" />
              {label}
              {key === 'WALLET' && walletBalance < grandTotal && (
                <Badge variant="destructive" className="ml-auto text-xs">Không đủ số dư</Badge>
              )}
            </button>
          ))}
        </CardContent>
      </Card>

      {/* Summary */}
      <Card>
        <CardContent className="space-y-1.5 py-4">
          <div className="flex justify-between text-sm text-muted-foreground">
            <span>Tạm tính</span><span>{formatVnd(subtotal)}</span>
          </div>
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
            * Phí giao hàng sẽ được tính chính xác khi đặt hàng.
          </p>
        </CardContent>
      </Card>

      <Button
        className="w-full"
        size="lg"
        onClick={handlePlaceOrder}
        disabled={placeOrder.isPending || walletInsufficient}
      >
        {placeOrder.isPending ? 'Đang đặt hàng…' : 'Đặt hàng'}
      </Button>
    </div>
  );
}
