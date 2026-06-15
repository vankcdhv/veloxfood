import type { NextConfig } from 'next';

// All /api/* traffic is forwarded to the Kong API gateway which handles
// per-service routing internally. Override GATEWAY_URL to point at a remote
// Kong instance (staging/prod). Services run on host ports 17080-17089; Kong
// proxy is published on host :17000.
// The previous per-service rewrites are preserved below as a reference comment.
const GATEWAY_URL = process.env.GATEWAY_URL ?? 'http://localhost:17000';

const nextConfig: NextConfig = {
  async redirects() {
    return [
      // Front door is the storefront, not the UI-kit scaffold at `/`.
      { source: '/', destination: '/shop', permanent: false },
      // Legacy links that pointed at non-existent pages.
      { source: '/shop/menu', destination: '/stores', permanent: false },
      { source: '/shop/cart', destination: '/cart', permanent: false },
    ];
  },
  async rewrites() {
    return [
      // Single catch-all: Next.js proxies /api/* same-origin to Kong.
      // Kong's declarative routes (backend/kong/kong.yml) reproduce the full
      // per-service routing that previously lived here.
      {
        source: '/api/:path*',
        destination: `${GATEWAY_URL}/api/:path*`,
      },
    ];
  },
};

export default nextConfig;
