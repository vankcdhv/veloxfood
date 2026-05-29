'use client';

import { useUsers } from '../hooks/use-users';

export function UserList() {
  const { data, isLoading, isError, error, refetch, isFetching } = useUsers();

  if (isLoading) {
    return <p className="text-muted-foreground text-sm">Đang tải users...</p>;
  }

  if (isError) {
    return (
      <div className="text-destructive space-y-2 text-sm">
        <p>Lỗi: {error instanceof Error ? error.message : 'Unknown error'}</p>
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
        {isFetching && <span>Đang refresh...</span>}
      </div>

      {users.length === 0 ? (
        <p className="text-muted-foreground text-sm">Chưa có user nào.</p>
      ) : (
        <ul className="divide-border divide-y rounded-md border">
          {users.map((u) => (
            <li key={u.id} className="flex items-center justify-between px-4 py-3">
              <div>
                <p className="text-sm font-medium">{u.full_name || '(không tên)'}</p>
                <p className="text-muted-foreground text-xs">{u.email}</p>
              </div>
              <span
                className={u.is_active ? 'text-xs text-green-600' : 'text-muted-foreground text-xs'}
              >
                {u.is_active ? 'active' : 'inactive'}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
