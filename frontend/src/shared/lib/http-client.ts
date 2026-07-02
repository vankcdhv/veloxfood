import axios, { isAxiosError, type AxiosInstance } from 'axios';
import { toast } from 'sonner';
import { env } from './env';
import { generateTraceId, TRACE_ID_HEADER } from './trace-id';
import { attachAuthRefreshInterceptor } from './auth-refresh';

export const http: AxiosInstance = axios.create({
  baseURL: env.API_BASE_URL,
  timeout: 10_000,
  // Send/receive the httpOnly auth cookies the backend issues (needed for
  // cross-origin prod; harmless same-origin via the Next dev proxy).
  withCredentials: true,
  headers: { 'Content-Type': 'application/json' },
});

http.interceptors.request.use((config) => {
  config.headers.set(TRACE_ID_HEADER, generateTraceId());
  return config;
});

// 401 → single-flight cookie refresh + retry.
attachAuthRefreshInterceptor(http);

// 429 (gateway rate limit) → one friendly toast instead of a raw error per
// caller. The id dedupes bursts: many parallel requests tripping the limit
// only surface a single notification.
http.interceptors.response.use(undefined, (error) => {
  if (isAxiosError(error) && error.response?.status === 429) {
    toast.error('Bạn thao tác quá nhanh. Vui lòng thử lại sau ít phút.', { id: 'rate-limited' });
  }
  return Promise.reject(error);
});
