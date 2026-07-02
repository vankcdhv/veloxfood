export const API_PREFIX = '/api/v1';

export const ROUTES = {
  home: '/',
  shop: {
    root: '/shop',
  },
  admin: {
    root: '/admin',
    users: '/admin/users',
    orders: '/admin/orders',
    shippers: '/admin/shippers',
    locations: '/admin/locations',
    stores: '/admin/stores',
    payouts: '/admin/payouts',
    incidents: '/admin/incidents',
    settings: '/admin/settings',
  },
  account: {
    profile: '/account/profile',
    favorites: '/account/favorites',
    locations: '/account/locations',
    store: '/account/store',
    wallet: '/account/wallet',
    orders: '/account/orders',
    orderDetail: (id: string) => `/account/orders/${id}`,
    deliveries: '/account/deliveries',
  },
  cart: '/cart',
  checkout: '/checkout',
  stores: {
    root: '/stores',
    detail: (id: string) => `/stores/${id}`,
  },
  auth: {
    login: '/login',
    register: '/register',
    forgotPassword: '/forgot-password',
    resetPassword: '/reset-password',
  },
  registerShipper: '/register-shipper',
  registerSeller: '/register-seller',
  support: {
    help: '/help',
    contact: '/contact',
  },
  search: '/search',
} as const;
