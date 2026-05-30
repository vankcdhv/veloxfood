'use client';

import { useEffect, useRef, useState } from 'react';
import { Loader2, Plus, Store as StoreIcon } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useAdminStores, useAdminStoreMutations } from '../hooks/use-stores';
import { SaleStatusBadge } from './sale-status-badge';
import { useUsers } from '@/features/users/hooks/use-users';
import type { AdminUser } from '@/features/users/types/user';
import type { Store } from '../types/store';

interface Props {
  onSelectStore: (store: Store) => void;
  selectedStoreId?: string;
}

export function AdminStoreList({ onSelectStore, selectedStoreId }: Props) {
  const { data: stores, isLoading, isError } = useAdminStores();
  const [showCreate, setShowCreate] = useState(false);

  if (isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-16 rounded-lg" />
        ))}
      </div>
    );
  }

  if (isError) {
    return <p className="text-destructive text-sm">Không tải được danh sách cửa hàng.</p>;
  }

  return (
    <>
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="font-semibold text-sm text-muted-foreground uppercase tracking-wider">
            Cửa hàng ({stores?.length ?? 0})
          </h2>
          <Button size="sm" variant="outline" onClick={() => setShowCreate(true)} className="gap-1">
            <Plus className="h-3.5 w-3.5" />
            Tạo mới
          </Button>
        </div>

        {(!stores || stores.length === 0) && (
          <div className="flex flex-col items-center gap-2 py-10 text-muted-foreground">
            <StoreIcon className="h-8 w-8 opacity-30" />
            <p className="text-sm">Chưa có cửa hàng nào.</p>
          </div>
        )}

        <div className="space-y-2">
          {stores?.map((store) => (
            <button
              key={store.ID}
              type="button"
              onClick={() => onSelectStore(store)}
              className={`w-full text-left rounded-lg border border-border px-4 py-3 transition-colors hover:bg-accent/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
                selectedStoreId === store.ID ? 'bg-primary/5 border-primary/40' : ''
              }`}
            >
              <div className="flex items-center justify-between gap-2">
                <span className="font-medium text-sm truncate">{store.Name}</span>
                <SaleStatusBadge status={store.SaleStatus} />
              </div>
              {store.BusinessType && (
                <p className="text-muted-foreground text-xs mt-0.5">{store.BusinessType}</p>
              )}
            </button>
          ))}
        </div>
      </div>

      {/* key remounts the dialog each open, resetting all state and generating a fresh vendor_id */}
      <CreateStoreDialog key={showCreate ? 'open' : 'closed'} open={showCreate} onClose={() => setShowCreate(false)} />
    </>
  );
}

// ---------- Create Store Dialog ----------

function CreateStoreDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { create } = useAdminStoreMutations();

  // Vendor ID generated once on mount (component is remounted via key each dialog open).
  const [vendorId] = useState(() => crypto.randomUUID());
  const [name, setName] = useState('');

  // Owner picker state
  const [ownerSearch, setOwnerSearch] = useState('');
  const [selectedOwner, setSelectedOwner] = useState<AdminUser | null>(null);
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Fetch active VENDOR_OWNER users matching search; only fires when there is input.
  const { data: userPage, isFetching: fetchingUsers } = useUsers({
    search: ownerSearch,
    status: 'active',
    role_id: '754a8e4d-4e3b-4241-ba40-619f411aba3b',
    page: 1,
    page_size: 10,
  });
  const suggestions = userPage?.items ?? [];

  // Close dropdown on outside click.
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setDropdownOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, []);

  const handleSearchChange = (v: string) => {
    setOwnerSearch(v);
    setSelectedOwner(null);
    setDropdownOpen(v.length > 0);
  };

  const pickOwner = (u: AdminUser) => {
    setSelectedOwner(u);
    setOwnerSearch(`${u.full_name} · ${u.email ?? ''}`);
    setDropdownOpen(false);
  };

  const submit = () => {
    if (!selectedOwner) {
      toast.error('Vui lòng chọn chủ cửa hàng.');
      return;
    }
    if (!name.trim()) {
      toast.error('Vui lòng nhập tên cửa hàng.');
      return;
    }
    create.mutate(
      { vendor_id: vendorId, owner_user_id: selectedOwner.id, name: name.trim() },
      {
        onSuccess: () => {
          toast.success('Đã tạo cửa hàng.');
          onClose();
        },
        onError: (e) => toast.error(getApiErrorMessage(e, 'Tạo cửa hàng thất bại')),
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Tạo cửa hàng mới</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          {/* Owner user picker — only VENDOR_OWNER accounts */}
          <div>
            <label className="text-foreground mb-1.5 block text-sm font-medium">
              Chủ cửa hàng
            </label>
            <p className="text-muted-foreground mb-1.5 text-xs">
              Chỉ hiển thị tài khoản có vai trò Chủ cửa hàng (vendor).
            </p>
            <div className="relative" ref={dropdownRef}>
              <Input
                value={ownerSearch}
                onChange={(e) => handleSearchChange(e.target.value)}
                onFocus={() => ownerSearch.length > 0 && setDropdownOpen(true)}
                placeholder="Tìm theo tên hoặc email..."
              />
              {fetchingUsers && (
                <Loader2 className="absolute right-2 top-2.5 h-4 w-4 animate-spin text-muted-foreground" />
              )}
              {dropdownOpen && suggestions.length > 0 && (
                <ul className="absolute z-50 mt-1 w-full rounded-md border bg-popover shadow-md max-h-48 overflow-auto">
                  {suggestions.map((u) => (
                    <li key={u.id}>
                      <button
                        type="button"
                        className="w-full px-3 py-2 text-left text-sm hover:bg-accent"
                        onMouseDown={() => pickOwner(u)}
                      >
                        <span className="font-medium">{u.full_name}</span>
                        <span className="text-muted-foreground ml-1">· {u.email ?? '—'}</span>
                      </button>
                    </li>
                  ))}
                </ul>
              )}
              {dropdownOpen && !fetchingUsers && ownerSearch.length > 0 && suggestions.length === 0 && (
                <div className="absolute z-50 mt-1 w-full rounded-md border bg-popover px-3 py-2 text-sm text-muted-foreground shadow-md">
                  Không tìm thấy người dùng nào.
                </div>
              )}
            </div>
          </div>

          {/* Store name */}
          <div>
            <label className="text-foreground mb-1.5 block text-sm font-medium">Tên cửa hàng</label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="VD: Bếp nhà Lan"
            />
          </div>

          {/* Vendor ID is generated automatically for this manual bypass; the
              value is irrelevant to the admin, so we don't surface the UUID. */}
          <p className="text-muted-foreground text-xs">
            Mã vendor sẽ được tạo tự động khi lưu (tạo cửa hàng thủ công).
          </p>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={create.isPending}>
            Huỷ
          </Button>
          <Button onClick={submit} disabled={create.isPending}>
            {create.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Tạo cửa hàng'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---------- Admin Store Detail Card ----------

// Inline store detail card for admin — shows key fields, owner name instead of raw UUID.
export function AdminStoreDetailCard({ store }: { store: Store }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center gap-2">
          {store.Name}
          <SaleStatusBadge status={store.SaleStatus} />
        </CardTitle>
      </CardHeader>
      <CardContent className="text-sm space-y-1 text-muted-foreground">
        {store.BusinessType && <p>Loại hình: {store.BusinessType}</p>}
        {store.Address && <p>Địa chỉ: {store.Address}</p>}
        {store.Phone && <p>SĐT: {store.Phone}</p>}
        <p>Chủ cửa hàng: {store.OwnerUserName || '—'}</p>
      </CardContent>
    </Card>
  );
}
