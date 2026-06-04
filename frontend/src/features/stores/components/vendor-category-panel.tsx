'use client';

import { useState } from 'react';
import { Loader2, Plus, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { ConfirmDialog } from '@/shared/ui/confirm-dialog';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useStoreMenu, useVendorStoreMutations } from '../hooks/use-stores';
import type { Category } from '../types/store';

interface Props {
  storeId: string;
}

export function VendorCategoryPanel({ storeId }: Props) {
  const { data: menu, isLoading } = useStoreMenu(storeId);
  const m = useVendorStoreMutations(storeId);
  const [newName, setNewName] = useState('');
  const [pendingDelete, setPendingDelete] = useState<Category | null>(null);

  const categories: Category[] = menu?.map((mc) => mc.Category) ?? [];

  const addCategory = () => {
    const name = newName.trim();
    if (!name) {
      toast.error('Vui lòng nhập tên danh mục.');
      return;
    }
    m.createCategory.mutate(
      { name, sortOrder: categories.length },
      {
        onSuccess: () => { toast.success('Đã thêm danh mục.'); setNewName(''); },
        onError: (e) => toast.error(getApiErrorMessage(e, 'Thêm danh mục thất bại')),
      },
    );
  };

  const confirmDelete = () => {
    if (!pendingDelete) return;
    m.deleteCategory.mutate(pendingDelete.ID, {
      onSuccess: () => { toast.success('Đã xoá danh mục.'); setPendingDelete(null); },
      onError: (e) => { toast.error(getApiErrorMessage(e, 'Xoá thất bại')); setPendingDelete(null); },
    });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Danh mục</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {isLoading && <Skeleton className="h-20" />}
        {!isLoading && categories.length === 0 && (
          <p className="text-muted-foreground text-sm">Chưa có danh mục nào.</p>
        )}
        <ul className="space-y-1">
          {categories.map((cat) => (
            <CategoryRow
              key={cat.ID}
              category={cat}
              onDelete={() => setPendingDelete(cat)}
              deleting={m.deleteCategory.isPending}
            />
          ))}
        </ul>
        {/* Add new category */}
        <div className="flex gap-2 pt-1">
          <Input
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && addCategory()}
            placeholder="Tên danh mục mới"
            disabled={m.createCategory.isPending}
          />
          <Button
            size="icon"
            onClick={addCategory}
            disabled={m.createCategory.isPending}
            aria-label="Thêm danh mục"
          >
            {m.createCategory.isPending
              ? <Loader2 className="h-4 w-4 animate-spin" />
              : <Plus className="h-4 w-4" />}
          </Button>
        </div>
      </CardContent>

      <ConfirmDialog
        open={!!pendingDelete}
        onOpenChange={(o) => !o && setPendingDelete(null)}
        title="Xoá danh mục?"
        description={`Xoá danh mục "${pendingDelete?.Name ?? ''}". Các món trong danh mục sẽ chuyển về nhóm "Khác".`}
        confirmLabel="Xoá"
        destructive
        loading={m.deleteCategory.isPending}
        onConfirm={confirmDelete}
      />
    </Card>
  );
}

function CategoryRow({
  category, onDelete, deleting,
}: {
  category: Category;
  onDelete: () => void;
  deleting: boolean;
}) {
  return (
    <li className="group flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-accent/50 transition-colors">
      <span className="flex-1 text-sm truncate">{category.Name}</span>
      <button
        type="button"
        aria-label="Xoá danh mục"
        onClick={onDelete}
        disabled={deleting}
        className="text-muted-foreground hover:text-destructive opacity-60 group-hover:opacity-100 transition-opacity p-1 rounded disabled:opacity-30"
      >
        <Trash2 className="h-4 w-4" />
      </button>
    </li>
  );
}
