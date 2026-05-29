'use client';

import { createContext, useContext, type ReactNode } from 'react';
import { useSession, type SessionState } from '../hooks/use-session';

const AuthContext = createContext<SessionState | null>(null);

// AuthProvider runs a single useSession subscription and shares it via context
// so components read auth state without each spawning a query (React Query
// dedupes by key anyway, but the context keeps the API explicit).
export function AuthProvider({ children }: { children: ReactNode }) {
  const session = useSession();
  return <AuthContext.Provider value={session}>{children}</AuthContext.Provider>;
}

export function useAuth(): SessionState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within <AuthProvider>');
  return ctx;
}
