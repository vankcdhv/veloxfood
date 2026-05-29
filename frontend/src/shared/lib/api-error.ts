import { AxiosError } from 'axios';
import type { ApiResponse } from '@/shared/api/api-response';

// getApiErrorMessage extracts a human message from a failed request, preferring
// the backend envelope's `error`/`message` fields over the raw axios message.
export function getApiErrorMessage(
  err: unknown,
  fallback = 'Đã có lỗi xảy ra, vui lòng thử lại.',
): string {
  if (err instanceof AxiosError) {
    const data = err.response?.data as ApiResponse | undefined;
    if (data?.error) return data.error;
    if (data?.message) return data.message;
    if (err.message) return err.message;
  }
  if (err instanceof Error && err.message) return err.message;
  return fallback;
}
