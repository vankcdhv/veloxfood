import type { ReactNode } from 'react';
import { BrandMark } from '@/widgets/brand-mark/brand-mark';
import { AuthHero } from '@/widgets/auth-hero/auth-hero';
import { ThemeToggle } from '@/shared/ui/theme-toggle';
import { ROUTES } from '@/shared/config/constants';

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="bg-background grid min-h-dvh lg:grid-cols-2">
      <AuthHero />

      <main className="relative flex min-h-dvh flex-col px-4 py-6 sm:px-6 sm:py-10">
        <div className="flex items-center justify-between">
          <BrandMark href={ROUTES.home} className="lg:invisible" />
          <ThemeToggle />
        </div>

        <div className="flex flex-1 items-center justify-center py-8">
          <div className="w-full max-w-md">{children}</div>
        </div>

        <footer className="text-muted-foreground text-center text-xs">
          © {new Date().getFullYear()} VeloxFood · Món ngon mỗi ngày, giao tận tay bạn.
        </footer>
      </main>
    </div>
  );
}
