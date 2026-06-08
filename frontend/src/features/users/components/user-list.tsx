'use client';

import { useEffect, useState } from 'react';
import { Search, Ban, RotateCcw } from 'lucide-react';
import { toast } from 'sonner';
import { Input } from '@/shared/ui/input';
import { Button } from '@/shared/ui/button';
import { Badge } from '@/shared/ui/badge';
import { ConfirmDialog } from '@/shared/ui/confirm-dialog';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useUsers, useSuspendUser, useReactivateUser } from '../hooks/use-users';
import type { AdminUser, UserStatus } from '../types/user';

const STATUS_META: Record<UserStatus, { label: string; variant: 'success' | 'destructive' | 'warning' }> = {
  active: { label: 'Hoạt động', variant: 'success' },
  suspended: { label: 'Bị khoá', variant: 'destructive' },
  pending: { label: 'Chờ duyệt', variant: 'warning' },
};

const STATUS_FILTERS: { value: string; label: string }[] = [
  { value: '', label: 'Tất cả trạng thái' },
  { value: 'active', label: 'Hoạt động' },
  { value: 'suspended', label: 'Bị khoá' },
  { value: 'pending', label: 'Chờ duyệt' },
];

export function UserList() {
  const [searchInput, setSearchInput] = useState('');
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [pending, setPending] = useState<{ user: AdminUser; action: 'suspend' | 'reactivate' } | null>(null);

  // Debounce the search box.
  useEffect(() => {
    const t = setTimeout(() => setSearch(searchInput.trim()), 300);
    return () => clearTimeout(t);
  }, [searchInput]);

  const { data, isLoading, isError, error, refetch, isFetching } = useUsers({
    search: search || undefined,
    status: status || undefined,
    page: 1,
    page_size: 50,
  });
  const suspend = useSuspendUser();
  const reactivate = useReactivateUser();
  const busy = suspend.isPending || reactivate.isPending;

  const confirmAction = () => {
    if (!pending) return;
    const opts = {
      onSuccess: () => { toast.success(pending.action === 'suspend' ? 'Đã khoá người dùng.' : 'Đã mở khoá người dùng.'); setPending(null); },
      onError: (e: unknown) => { toast.error(getApiErrorMessage(e, 'Thao tác thất bại')); setPending(null); },
    };
    if (pending.action === 'suspend') suspend.mutate({ userId: pending.user.id }, opts);
    else reactivate.mutate(pending.user.id, opts);
  };

  return (
    <div className="space-y-4">
      {/* Filters */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="text-muted-foreground pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2" />
          <Input
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            placeholder="Tìm theo tên / email / SĐT…"
            className="pl-9"
          />
        </div>
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          className="border-input bg-background h-9 rounded-md border px-3 text-sm shrink-0"
        >
          {STATUS_FILTERS.map((f) => (
            <option key={f.value} value={f.value}>{f.label}</option>
          ))}
        </select>
      </div>

      <div className="text-muted-foreground flex items-center justify-between text-xs">
        <span>Tổng: {data?.total ?? 0}</span>
        {isFetching && <span>Đang tải…</span>}
      </div>

      {isLoading && <p className="text-muted-foreground text-sm">Đang tải người dùng…</p>}

      {isError && (
        <div className="text-destructive space-y-2 text-sm">
          <p>Lỗi: {error instanceof Error ? error.message : 'Không tải được danh sách'}</p>
          <button type="button" onClick={() => refetch()} className="text-primary underline">Thử lại</button>
        </div>
      )}

      {!isLoading && !isError && (data?.items?.length ?? 0) === 0 && (
        <p className="text-muted-foreground text-sm py-6 text-center">Không tìm thấy người dùng nào.</p>
      )}

      {!isLoading && !isError && (data?.items?.length ?? 0) > 0 && (
        <ul className="divide-border divide-y rounded-md border">
          {data!.items.map((u) => (
            <li key={u.id} className="flex items-center justify-between gap-3 px-4 py-3">
              <div className="min-w-0">
                <p className="text-sm font-medium truncate">{u.full_name || '(không tên)'}</p>
                <p className="text-muted-foreground text-xs truncate">{u.email ?? u.phone ?? '—'}</p>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <Badge variant={STATUS_META[u.status]?.variant ?? 'outline'}>
                  {STATUS_META[u.status]?.label ?? u.status}
                </Badge>
                {u.status === 'active' && (
                  <Button size="sm" variant="outline" disabled={busy}
                    className="text-destructive border-destructive/50 hover:bg-destructive/5"
                    onClick={() => setPending({ user: u, action: 'suspend' })}>
                    <Ban className="h-3.5 w-3.5 mr-1" /> Khoá
                  </Button>
                )}
                {u.status === 'suspended' && (
                  <Button size="sm" variant="outline" disabled={busy}
                    onClick={() => setPending({ user: u, action: 'reactivate' })}>
                    <RotateCcw className="h-3.5 w-3.5 mr-1" /> Mở khoá
                  </Button>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}

      <ConfirmDialog
        open={!!pending}
        onOpenChange={(o) => !o && setPending(null)}
        title={pending?.action === 'suspend' ? 'Khoá người dùng?' : 'Mở khoá người dùng?'}
        description={
          pending?.action === 'suspend'
            ? `Người dùng "${pending?.user.full_name || pending?.user.email}" sẽ bị khoá và không đăng nhập được.`
            : `Mở khoá cho "${pending?.user.full_name || pending?.user.email}"?`
        }
        confirmLabel={pending?.action === 'suspend' ? 'Khoá' : 'Mở khoá'}
        destructive={pending?.action === 'suspend'}
        loading={busy}
        onConfirm={confirmAction}
      />
    </div>
  );
}
