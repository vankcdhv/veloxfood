import { NextResponse, type NextRequest } from 'next/server';

// Presence of the refresh_token cookie (7d, Path=/) signals a live session.
// We deliberately check refresh (not access, which expires in 15m) so a user
// whose access token lapsed but can still refresh is not bounced to login.
const SESSION_COOKIE = 'refresh_token';

const AUTH_PAGES = ['/login', '/register'];

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const hasSession = request.cookies.has(SESSION_COOKIE);

  // Protect the admin area — anonymous users go to login with a return path.
  if (pathname.startsWith('/admin') && !hasSession) {
    const url = request.nextUrl.clone();
    url.pathname = '/login';
    url.searchParams.set('next', pathname);
    return NextResponse.redirect(url);
  }

  // Authenticated users have no reason to see login/register.
  if (hasSession && AUTH_PAGES.includes(pathname)) {
    const url = request.nextUrl.clone();
    url.pathname = '/';
    url.search = '';
    return NextResponse.redirect(url);
  }

  return NextResponse.next();
}

export const config = {
  matcher: ['/admin/:path*', '/login', '/register'],
};
