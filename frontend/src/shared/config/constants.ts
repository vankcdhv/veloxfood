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
  },
  auth: {
    login: '/login',
    register: '/register',
    forgotPassword: '/forgot-password',
    resetPassword: '/reset-password',
  },
} as const;
