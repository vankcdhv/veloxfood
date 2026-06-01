import type { NextConfig } from 'next';

// All /api/* traffic is forwarded to the Kong API gateway which handles
// per-service routing internally. Override GATEWAY_URL to point at a remote
// Kong instance (staging/prod). Services run on host ports 8080-8089; Kong
// proxy listens on :8000. The previous per-service rewrites are preserved
// below as a reference comment — Kong reproduces the same routing logic.
const GATEWAY_URL = process.env.GATEWAY_URL ?? 'http://localhost:8000';

const nextConfig: NextConfig = {
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
