import type { NextConfig } from 'next';

// Dev: proxy /api/* to the Go backends per service (avoid CORS, same-origin).
// Until the Kong gateway is in place, route by path prefix to each service.
// Prod: a gateway (Kong) routes /api → services. Override via env.
const USER_API_URL = process.env.USER_API_URL ?? 'http://localhost:8080';
const LOCATION_API_URL = process.env.LOCATION_API_URL ?? 'http://localhost:8081';
const STORE_API_URL = process.env.STORE_API_URL ?? 'http://localhost:8082';

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      // Location service (must precede the catch-all).
      { source: '/api/v1/locations/:path*', destination: `${LOCATION_API_URL}/api/v1/locations/:path*` },
      { source: '/api/v1/me/locations/:path*', destination: `${LOCATION_API_URL}/api/v1/me/locations/:path*` },
      { source: '/api/v1/admin/buildings/:path*', destination: `${LOCATION_API_URL}/api/v1/admin/buildings/:path*` },
      { source: '/api/v1/admin/floors/:path*', destination: `${LOCATION_API_URL}/api/v1/admin/floors/:path*` },
      { source: '/api/v1/admin/rooms/:path*', destination: `${LOCATION_API_URL}/api/v1/admin/rooms/:path*` },
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
