import type { NextConfig } from 'next';

// Dev: proxy /api/* to the Go backends per service (avoid CORS, same-origin).
// Until the Kong gateway is in place, route by path prefix to each service.
// Prod: a gateway (Kong) routes /api → services. Override via env.
const USER_API_URL = process.env.USER_API_URL ?? 'http://localhost:8080';
const LOCATION_API_URL = process.env.LOCATION_API_URL ?? 'http://localhost:8081';
const STORE_API_URL = process.env.STORE_API_URL ?? 'http://localhost:8082';
const PROMOTION_API_URL = process.env.PROMOTION_API_URL ?? 'http://localhost:8084';
const PAYMENT_API_URL = process.env.PAYMENT_API_URL ?? 'http://localhost:8085';

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      // Location service (must precede the catch-all).
      { source: '/api/v1/locations/:path*', destination: `${LOCATION_API_URL}/api/v1/locations/:path*` },
      { source: '/api/v1/me/locations/:path*', destination: `${LOCATION_API_URL}/api/v1/me/locations/:path*` },
      { source: '/api/v1/admin/buildings/:path*', destination: `${LOCATION_API_URL}/api/v1/admin/buildings/:path*` },
      { source: '/api/v1/admin/floors/:path*', destination: `${LOCATION_API_URL}/api/v1/admin/floors/:path*` },
      { source: '/api/v1/admin/rooms/:path*', destination: `${LOCATION_API_URL}/api/v1/admin/rooms/:path*` },
      // Payment service — wallet routes must precede the catch-all user rule.
      { source: '/api/v1/me/wallet', destination: `${PAYMENT_API_URL}/api/v1/me/wallet` },
      { source: '/api/v1/wallet/:path*', destination: `${PAYMENT_API_URL}/api/v1/wallet/:path*` },
      { source: '/api/v1/payments/:path*', destination: `${PAYMENT_API_URL}/api/v1/payments/:path*` },
      // Promotion service — more specific store-scoped and top-level routes must
      // precede the generic /stores/:path* → STORE rule to avoid mis-routing.
      { source: '/api/v1/stores/:storeId/promotions/:path*', destination: `${PROMOTION_API_URL}/api/v1/stores/:storeId/promotions/:path*` },
      { source: '/api/v1/stores/:storeId/promotions', destination: `${PROMOTION_API_URL}/api/v1/stores/:storeId/promotions` },
      { source: '/api/v1/promotions/validate', destination: `${PROMOTION_API_URL}/api/v1/promotions/validate` },
      // Store & Catalog service (must precede the catch-all).
      { source: '/api/v1/stores/:path*', destination: `${STORE_API_URL}/api/v1/stores/:path*` },
      { source: '/api/v1/categories/:path*', destination: `${STORE_API_URL}/api/v1/categories/:path*` },
      { source: '/api/v1/options/:path*', destination: `${STORE_API_URL}/api/v1/options/:path*` },
      { source: '/api/v1/combos/:path*', destination: `${STORE_API_URL}/api/v1/combos/:path*` },
      { source: '/api/v1/admin/stores/:path*', destination: `${STORE_API_URL}/api/v1/admin/stores/:path*` },
      // Everything else → user service.
      { source: '/api/:path*', destination: `${USER_API_URL}/api/:path*` },
    ];
  },
};

export default nextConfig;
