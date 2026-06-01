/**
 * Format an integer VND amount with Vietnamese thousands separators + đ suffix.
 * Example: 150000 → "150.000đ"
 */
export function formatVnd(amount: number): string {
  return `${amount.toLocaleString('vi-VN')}đ`;
}
