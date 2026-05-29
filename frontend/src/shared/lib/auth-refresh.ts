import type { AxiosError, AxiosInstance, InternalAxiosRequestConfig } from 'axios';
import { API_PREFIX } from '@/shared/config/constants';

const REFRESH_PATH = `${API_PREFIX}/auth/refresh`;
const AUTH_PREFIX = `${API_PREFIX}/auth/`;

type RetriableConfig = InternalAxiosRequestConfig & { _retry?: boolean };

// Single-flight refresh: concurrent 401s share one /auth/refresh call so the
// single-use refresh token is not rotated more than once per expiry window.
let refreshPromise: Promise<void> | null = null;

function runRefresh(http: AxiosInstance): Promise<void> {
  if (!refreshPromise) {
    refreshPromise = http
      .post(REFRESH_PATH)
      .then(() => undefined)
      .finally(() => {
        refreshPromise = null;
      });
  }
  return refreshPromise;
}

// attachAuthRefreshInterceptor wires a response interceptor that, on a 401 from
// a non-auth endpoint, transparently refreshes the cookie pair once and retries
// the original request. Auth endpoints (login/refresh/logout/...) are excluded
// so their own 401s surface to the caller.
export function attachAuthRefreshInterceptor(http: AxiosInstance): void {
  http.interceptors.response.use(
    (res) => res,
    async (error: AxiosError) => {
      const status = error.response?.status;
      const config = error.config as RetriableConfig | undefined;

      if (status !== 401 || !config) return Promise.reject(error);

      const url = config.url ?? '';
      if (url.includes(AUTH_PREFIX)) return Promise.reject(error);
      if (config._retry) return Promise.reject(error);
      config._retry = true;

      try {
        await runRefresh(http);
      } catch {
        // Refresh failed → session is gone; let the original 401 propagate.
        return Promise.reject(error);
      }
      return http(config);
    },
  );
}
