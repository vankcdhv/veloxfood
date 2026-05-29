import axios, { type AxiosInstance } from 'axios';
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
