export const API_PREFIX = '/api/v1';

export const ROUTES = {
  home: '/',
  shop: {
    root: '/shop',
    menu: '/shop/menu',
    cart: '/shop/cart',
  },
  admin: {
    root: '/admin',
    users: '/admin/users',
    orders: '/admin/orders',
    products: '/admin/products',
    shippers: '/admin/shippers',
    locations: '/admin/locations',
    stores: '/admin/stores',
  },
  account: {
    locations: '/account/locations',
    store: '/account/store',
    wallet: '/account/wallet',
    orders: '/account/orders',
    orderDetail: (id: string) => `/account/orders/${id}`,
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
} as const;
