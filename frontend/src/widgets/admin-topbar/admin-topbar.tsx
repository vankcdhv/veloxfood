'use client';

import { Bell, Search } from 'lucide-react';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { ThemeToggle } from '@/shared/ui/theme-toggle';
import { UserMenu } from '@/features/auth/components/user-menu';

interface AdminTopbarProps {
  title?: string;
  description?: string;
}

export function AdminTopbar({ title, description }: AdminTopbarProps) {
  return (
    <header className="border-border bg-background/80 sticky top-0 z-30 flex h-16 items-center gap-4 border-b px-4 backdrop-blur-md sm:px-6">
      <div className="min-w-0 flex-1">
        {title && <h1 className="font-serif text-lg font-semibold tracking-tight">{title}</h1>}
        {description && <p className="text-muted-foreground truncate text-xs">{description}</p>}
      </div>

      <div className="hidden items-center gap-2 sm:flex">
        <div className="relative">
          <Search className="text-muted-foreground pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2" />
          <Input
            type="search"
            placeholder="Tìm kiếm..."
            className="w-56 pl-9"
            aria-label="Tìm kiếm trong dashboard"
          />
        </div>
      </div>

      <div className="flex items-center gap-1">
        <Button variant="ghost" size="icon" aria-label="Thông báo">
          <Bell className="h-4 w-4" />
        </Button>
        <ThemeToggle />
        <UserMenu />
      </div>
    </header>
  );
}
