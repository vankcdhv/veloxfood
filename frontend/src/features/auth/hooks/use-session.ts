'use client';

import { useQuery } from '@tanstack/react-query';
import { getMe } from '../api/auth-api';
import type { MeResponse } from '../types/auth';

export const ME_QUERY_KEY = ['auth', 'me'] as const;

// Roles that may access the /admin area.
const ADMIN_ROLE_CODES = new Set(['SUPER_ADMIN', 'SCHOOL_ADMIN']);

export interface SessionState {
  data: MeResponse | undefined;
  user: MeResponse['user'] | null;
  roles: MeResponse['roles'];
  memberships: MeResponse['vendor_memberships'];
  isLoading: boolean;
  isAuthenticated: boolean;
  isAdmin: boolean;
  refetch: () => void;
}

// useSession is the single source of truth for the current session. Tokens live
// in httpOnly cookies (JS cannot read them) so the session is derived from /me.
export function useSession(): SessionState {
  const q = useQuery({
    queryKey: ME_QUERY_KEY,
    queryFn: getMe,
    retry: false,
    staleTime: 5 * 60_000,
  });

  const roles = q.data?.roles ?? [];

  return {
    data: q.data,
    user: q.data?.user ?? null,
    roles,
    memberships: q.data?.vendor_memberships ?? [],
    isLoading: q.isLoading,
    isAuthenticated: !!q.data && !q.isError,
    isAdmin: roles.some((r) => r.scope_type === 'global' && ADMIN_ROLE_CODES.has(r.role_code)),
    refetch: () => void q.refetch(),
  };
}
