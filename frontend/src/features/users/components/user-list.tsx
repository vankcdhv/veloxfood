'use client';

import { useUsers } from '../hooks/use-users';
import type { UserStatus } from '../types/user';

const STATUS_LABEL: Record<UserStatus, string> = {
  active: 'Hoạt động',
  suspended: 'Bị khóa',
  pending: 'Chờ duyệt',
};

const STATUS_CLASS: Record<UserStatus, string> = {
  active: 'text-green-600',
  suspended: 'text-destructive',
  pending: 'text-yellow-600',
};

export function UserList() {
  const { data, isLoading, isError, error, refetch, isFetching } = useUsers();

  if (isLoading) {
    return <p className="text-muted-foreground text-sm">Đang tải người dùng…</p>;
  }

  if (isError) {
    return (
      <div className="text-destructive space-y-2 text-sm">
        <p>Lỗi: {error instanceof Error ? error.message : 'Không tải được danh sách'}</p>
        <button type="button" onClick={() => refetch()} className="text-primary underline">
          Thử lại
        </button>
      </div>
    );
  }

  const users = data?.items ?? [];

  return (
    <div className="space-y-4">
      <div className="text-muted-foreground flex items-center justify-between text-xs">
        <span>
          Tổng: {data?.total ?? 0} · Trang: {data?.page ?? 1}
        </span>
        {isFetching && <span>Đang tải lại…</span>}
      </div>

      {users.length === 0 ? (
        <p className="text-muted-foreground text-sm">Chưa có người dùng nào.</p>
      ) : (
        <ul className="divide-border divide-y rounded-md border">
          {users.map((u) => (
            <li key={u.id} className="flex items-center justify-between px-4 py-3">
              <div>
                <p className="text-sm font-medium">{u.full_name || '(không tên)'}</p>
                <p className="text-muted-foreground text-xs">{u.email ?? '—'}</p>
              </div>
              <span className={STATUS_CLASS[u.status] ?? 'text-muted-foreground text-xs'}>
                {STATUS_LABEL[u.status] ?? u.status}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
