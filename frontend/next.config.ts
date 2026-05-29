import type { NextConfig } from 'next';

// Dev: proxy /api/* tới backend Go (avoid CORS, frontend gọi same-origin).
// Prod: thường có reverse proxy (nginx/ingress) lo route /api → backend, hoặc
// frontend được serve cùng domain với backend. Override bằng env BACKEND_API_URL.
const BACKEND_API_URL = process.env.BACKEND_API_URL ?? 'http://localhost:8080';

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${BACKEND_API_URL}/api/:path*`,
      },
    ];
  },
};

export default nextConfig;
