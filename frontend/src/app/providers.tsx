'use client';

import { useState, type ReactNode } from 'react';
import { QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { createQueryClient } from '@/shared/api/query-client';
import { ThemeProvider } from '@/shared/providers/theme-provider';
import { AuthProvider } from '@/features/auth/context/auth-provider';
import { Toaster } from '@/shared/ui/sonner';

export function Providers({ children }: { children: ReactNode }) {
  const [client] = useState(createQueryClient);

  return (
    <ThemeProvider>
      <QueryClientProvider client={client}>
        <AuthProvider>
          {children}
          <Toaster />
          {process.env.NODE_ENV === 'development' && (
            <ReactQueryDevtools initialIsOpen={false} buttonPosition="bottom-right" />
          )}
        </AuthProvider>
      </QueryClientProvider>
    </ThemeProvider>
  );
}
