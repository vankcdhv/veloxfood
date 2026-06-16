// Maps the promotion service's English ErrorReason strings (from
// services/promotion/internal/usecase/promotion_usecase.go) to Vietnamese
// copy for end users. Falls back to a generic message for unknown reasons.
const REASON_VI: Record<string, string> = {
  'promotion not found': 'Không tìm thấy mã giảm giá.',
  'promotion does not belong to this store': 'Mã không áp dụng cho cửa hàng này.',
  'promotion is not active': 'Mã hiện không khả dụng.',
  'promotion has not started yet': 'Mã chưa tới thời gian áp dụng.',
  'promotion has expired': 'Mã đã hết hạn.',
  'order subtotal does not meet the minimum order requirement':
    'Đơn chưa đạt giá trị tối thiểu để dùng mã.',
  'promotion usage limit has been reached': 'Mã đã hết lượt sử dụng.',
};

export function promotionErrorVi(reason?: string): string {
  if (!reason) return 'Mã không hợp lệ.';
  return REASON_VI[reason] ?? 'Mã không hợp lệ.';
}
