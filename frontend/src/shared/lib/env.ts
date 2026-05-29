// API_BASE_URL = '' (default) → axios gọi same-origin path /api/v1/...
// Next.js rewrites trong next.config.ts proxy sang backend Go ở dev.
// Override khi deploy frontend tách riêng backend: NEXT_PUBLIC_API_BASE_URL=https://api.example.com
const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? '';

export const env = {
  API_BASE_URL,
} as const;

export type Env = typeof env;
