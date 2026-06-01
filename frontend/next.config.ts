import type { NextConfig } from 'next';

// Dev: proxy /api/* to the Go backends per service (avoid CORS, same-origin).
// Until the Kong gateway is in place, route by path prefix to each service.
// Prod: a gateway (Kong) routes /api → services. Override via env.
const USER_API_URL = process.env.USER_API_URL ?? 'http://localhost:8080';
const LOCATION_API_URL = process.env.LOCATION_API_URL ?? 'http://localhost:8081';
const STORE_API_URL = process.env.STORE_API_URL ?? 'http://localhost:8082';
const ORDER_API_URL = process.env.ORDER_API_URL ?? 'http://localhost:8083';
const PROMOTION_API_URL = process.env.PROMOTION_API_URL ?? 'http://localhost:8084';
const PAYMENT_API_URL = process.env.PAYMENT_API_URL ?? 'http://localhost:8085';
const DELIVERY_API_URL = process.env.DELIVERY_API_URL ?? 'http://localhost:8086';
const REVIEW_API_URL = process.env.REVIEW_API_URL ?? 'http://localhost:8087';
const REPORTING_API_URL = process.env.REPORTING_API_URL ?? 'http://localhost:8088';
const NOTIFICATION_API_URL = process.env.NOTIFICATION_API_URL ?? 'http://localhost:8089';

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      // ── Phase D services ─────────────────────────────────────────────────────
      // These rules must precede any generic /api/v1/orders, /api/v1/stores,
      // and /api/v1/admin catch-alls to avoid mis-routing.

      // Reporting service — admin analytics and recent-orders summary.
      { source: '/api/v1/admin/analytics', destination: `${REPORTING_API_URL}/api/v1/admin/analytics` },
      { source: '/api/v1/admin/orders/recent', destination: `${REPORTING_API_URL}/api/v1/admin/orders/recent` },

      // Review service — order reviews, store reviews, admin review moderation.
      // /api/v1/orders/:id/reviews must precede the generic /api/v1/orders/:path* rule.
      // /api/v1/stores/:id/reviews must precede the generic /api/v1/stores/:path* rule.
      // /api/v1/admin/reviews must precede any generic /api/v1/admin/:path* rule.
      { source: '/api/v1/orders/:orderId/reviews', destination: `${REVIEW_API_URL}/api/v1/orders/:orderId/reviews` },
      { source: '/api/v1/stores/:storeId/reviews', destination: `${REVIEW_API_URL}/api/v1/stores/:storeId/reviews` },
      { source: '/api/v1/reviews/:path*', destination: `${REVIEW_API_URL}/api/v1/reviews/:path*` },
      { source: '/api/v1/admin/reviews/:path*', destination: `${REVIEW_API_URL}/api/v1/admin/reviews/:path*` },

      // Notification service — /me/notifications and /me/device-tokens.
      // Must precede the generic /api/* → user-service catch-all.
      { source: '/api/v1/me/notifications/unread-count', destination: `${NOTIFICATION_API_URL}/api/v1/me/notifications/unread-count` },
      { source: '/api/v1/me/notifications/:path*', destination: `${NOTIFICATION_API_URL}/api/v1/me/notifications/:path*` },
      { source: '/api/v1/me/notifications', destination: `${NOTIFICATION_API_URL}/api/v1/me/notifications` },
      { source: '/api/v1/me/device-tokens/:path*', destination: `${NOTIFICATION_API_URL}/api/v1/me/device-tokens/:path*` },
      { source: '/api/v1/me/device-tokens', destination: `${NOTIFICATION_API_URL}/api/v1/me/device-tokens` },

      // ── Existing services ─────────────────────────────────────────────────
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
      // Delivery service — /me/deliveries is distinct from /me/cart and /me/wallet.
      // Both /deliveries/* and /me/deliveries route to the delivery service.
      { source: '/api/v1/me/deliveries', destination: `${DELIVERY_API_URL}/api/v1/me/deliveries` },
      { source: '/api/v1/deliveries/:path*', destination: `${DELIVERY_API_URL}/api/v1/deliveries/:path*` },
      { source: '/api/v1/deliveries', destination: `${DELIVERY_API_URL}/api/v1/deliveries` },
      { source: '/api/v1/admin/deliveries', destination: `${DELIVERY_API_URL}/api/v1/admin/deliveries` },
      { source: '/api/v1/admin/incidents', destination: `${DELIVERY_API_URL}/api/v1/admin/incidents` },
      // Order service — cart and order routes (more specific store-scoped order
      // routes must precede the generic /stores/:path* → STORE rule).
      { source: '/api/v1/orders/:path*', destination: `${ORDER_API_URL}/api/v1/orders/:path*` },
      { source: '/api/v1/orders', destination: `${ORDER_API_URL}/api/v1/orders` },
      { source: '/api/v1/me/cart/:path*', destination: `${ORDER_API_URL}/api/v1/me/cart/:path*` },
      { source: '/api/v1/me/cart', destination: `${ORDER_API_URL}/api/v1/me/cart` },
      { source: '/api/v1/stores/:storeId/orders/:path*', destination: `${ORDER_API_URL}/api/v1/stores/:storeId/orders/:path*` },
      { source: '/api/v1/stores/:storeId/orders', destination: `${ORDER_API_URL}/api/v1/stores/:storeId/orders` },
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
