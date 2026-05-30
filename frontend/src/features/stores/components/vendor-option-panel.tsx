'use client';

import { useState } from 'react';
import { ChevronDown, ChevronRight, Loader2, Pencil, Plus, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { Skeleton } from '@/shared/ui/skeleton';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useOptionGroups, useOptionGroupMutations, useOptions } from '../hooks/use-stores';
import type { Option, OptionGroup } from '../types/store';

interface Props {
  storeId: string;
}

export function VendorOptionPanel({ storeId }: Props) {
  const { data: groups, isLoading, isError } = useOptionGroups(storeId);
  const m = useOptionGroupMutations(storeId);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [editGroup, setEditGroup] = useState<OptionGroup | null>(null);
  const [showCreate, setShowCreate] = useState(false);

  const deleteGroup = (ogId: string) => {
    m.deleteGroup.mutate(ogId, {
      onSuccess: () => toast.success('Đã xoá nhóm tuỳ chọn.'),
      onError: (e) => toast.error(getApiErrorMessage(e, 'Xoá thất bại')),
    });
  };

  return (
    <>
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold">Nhóm tuỳ chọn</h2>
          <Button size="sm" onClick={() => setShowCreate(true)} className="gap-1 h-8 text-xs">
            <Plus className="h-3.5 w-3.5" />
            Thêm nhóm
          </Button>
        </div>

        {isLoading && (
          <div className="space-y-2">
            {[1, 2].map((i) => <Skeleton key={i} className="h-14 rounded-xl" />)}
          </div>
        )}

        {isError && (
          <p className="text-destructive text-sm">Không tải được nhóm tuỳ chọn.</p>
        )}

        {!isLoading && !isError && (!groups || groups.length === 0) && (
          <p className="text-muted-foreground text-sm">Chưa có nhóm tuỳ chọn nào.</p>
        )}

        {groups?.map((group) => (
          <Card key={group.ID}>
            <CardHeader className="pb-2 cursor-pointer select-none" onClick={() => setExpanded(expanded === group.ID ? null : group.ID)}>
              <div className="flex items-center gap-2">
                {expanded === group.ID
                  ? <ChevronDown className="h-4 w-4 shrink-0 text-muted-foreground" />
                  : <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />}
                <CardTitle className="text-sm font-semibold flex-1">{group.Name}</CardTitle>
                <div className="flex items-center gap-1.5 shrink-0" onClick={(e) => e.stopPropagation()}>
                  <Badge variant="outline" className="text-xs">
                    {group.MinSelect}–{group.MaxSelect}
                  </Badge>
                  {group.Required && (
                    <Badge variant="default" className="text-xs">Bắt buộc</Badge>
                  )}
                  <button
                    type="button"
                    aria-label="Chỉnh sửa nhóm"
                    onClick={() => setEditGroup(group)}
                    className="text-muted-foreground hover:text-primary p-1"
                  >
                    <Pencil className="h-3.5 w-3.5" />
                  </button>
                  <button
                    type="button"
                    aria-label="Xoá nhóm"
                    onClick={() => deleteGroup(group.ID)}
                    disabled={m.deleteGroup.isPending}
                    className="text-muted-foreground hover:text-destructive p-1 disabled:opacity-30"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>
            </CardHeader>

            {expanded === group.ID && (
              <CardContent className="pt-0">
                <OptionList storeId={storeId} group={group} mutations={m} />
              </CardContent>
            )}
          </Card>
        ))}
      </div>

      <OptionGroupDialog
        storeId={storeId}
        open={showCreate}
        onClose={() => setShowCreate(false)}
        mutations={m}
      />

      <OptionGroupDialog
        key={editGroup?.ID ?? 'none'}
        storeId={storeId}
        open={!!editGroup}
        group={editGroup ?? undefined}
        onClose={() => setEditGroup(null)}
        mutations={m}
      />
    </>
  );
}

// ---------- Options list for an expanded group ----------

function OptionList({
  storeId, group, mutations,
}: {
  storeId: string;
  group: OptionGroup;
  mutations: ReturnType<typeof useOptionGroupMutations>;
}) {
  const { data: options, isLoading } = useOptions(storeId, group.ID);
  const [editOpt, setEditOpt] = useState<Option | null>(null);
  const [showAdd, setShowAdd] = useState(false);

  const deleteOption = (optId: string) => {
    mutations.deleteOption.mutate(
      { ogId: group.ID, optId },
      {
        onSuccess: () => toast.success('Đã xoá tuỳ chọn.'),
        onError: (e) => toast.error(getApiErrorMessage(e, 'Xoá thất bại')),
      },
    );
  };

  return (
    <div className="space-y-2 border-t border-border pt-3">
      {isLoading && <Skeleton className="h-10" />}
      {!isLoading && (!options || options.length === 0) && (
        <p className="text-muted-foreground text-xs">Chưa có tuỳ chọn nào.</p>
      )}
      <ul className="space-y-1">
        {options?.map((opt) => (
          <li
            key={opt.ID}
            className="group flex items-center gap-3 rounded-md px-2 py-1.5 hover:bg-accent/40 transition-colors"
          >
            <span className="flex-1 text-sm truncate">{opt.Name}</span>
            <span className="text-xs text-primary font-medium shrink-0">
              {opt.ExtraPrice > 0 ? `+${opt.ExtraPrice.toLocaleString('vi-VN')}đ` : 'Miễn phí'}
            </span>
            <button
              type="button"
              aria-label="Chỉnh sửa tuỳ chọn"
              onClick={() => setEditOpt(opt)}
              className="text-muted-foreground hover:text-primary opacity-0 group-hover:opacity-100 p-1 transition-opacity"
            >
              <Pencil className="h-3.5 w-3.5" />
            </button>
            <button
              type="button"
              aria-label="Xoá tuỳ chọn"
              onClick={() => deleteOption(opt.ID)}
              disabled={mutations.deleteOption.isPending}
              className="text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 p-1 transition-opacity disabled:opacity-30"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </li>
        ))}
      </ul>

      <Button
        variant="outline"
        size="sm"
        className="h-7 gap-1 text-xs"
        onClick={() => setShowAdd(true)}
      >
        <Plus className="h-3 w-3" />
        Thêm tuỳ chọn
      </Button>

      <OptionDialog
        open={showAdd}
        ogId={group.ID}
        onClose={() => setShowAdd(false)}
        mutations={mutations}
      />
      <OptionDialog
        key={editOpt?.ID ?? 'none-edit'}
        open={!!editOpt}
        ogId={group.ID}
        option={editOpt ?? undefined}
        onClose={() => setEditOpt(null)}
        mutations={mutations}
      />
    </div>
  );
}

// ---------- Create / Edit option group dialog ----------

function OptionGroupDialog({
  open, group, onClose, mutations,
}: {
  storeId: string;
  open: boolean;
  group?: OptionGroup;
  onClose: () => void;
  mutations: ReturnType<typeof useOptionGroupMutations>;
}) {
  const isEdit = !!group;
  const [name, setName] = useState(group?.Name ?? '');
  const [minSelect, setMinSelect] = useState(String(group?.MinSelect ?? 0));
  const [maxSelect, setMaxSelect] = useState(String(group?.MaxSelect ?? 1));
  const [required, setRequired] = useState(group?.Required ?? false);

  const isPending = mutations.createGroup.isPending || mutations.updateGroup.isPending;

  const submit = () => {
    const min = parseInt(minSelect, 10);
    const max = parseInt(maxSelect, 10);
    if (!name.trim() || isNaN(min) || isNaN(max) || min < 0 || max < min) {
      toast.error('Tên và giới hạn chọn không hợp lệ.');
      return;
    }
    const body = { name: name.trim(), min_select: min, max_select: max, required };

    if (isEdit && group) {
      mutations.updateGroup.mutate(
        { ogId: group.ID, body },
        {
          onSuccess: () => { toast.success('Đã cập nhật nhóm.'); onClose(); },
          onError: (e) => toast.error(getApiErrorMessage(e, 'Cập nhật thất bại')),
        },
      );
    } else {
      mutations.createGroup.mutate(body, {
        onSuccess: () => { toast.success('Đã thêm nhóm tuỳ chọn.'); onClose(); },
        onError: (e) => toast.error(getApiErrorMessage(e, 'Thêm thất bại')),
      });
    }
  };

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Chỉnh sửa nhóm tuỳ chọn' : 'Thêm nhóm tuỳ chọn'}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <Field label="Tên nhóm" value={name} onChange={setName} placeholder="VD: Topping, Kích cỡ" />
          <div className="grid grid-cols-2 gap-3">
            <Field label="Chọn tối thiểu" value={minSelect} onChange={setMinSelect} type="number" placeholder="0" />
            <Field label="Chọn tối đa" value={maxSelect} onChange={setMaxSelect} type="number" placeholder="1" />
          </div>
          <div className="flex items-center gap-2">
            <input
              id="required-chk"
              type="checkbox"
              checked={required}
              onChange={(e) => setRequired(e.target.checked)}
              className="h-4 w-4 rounded"
            />
            <label htmlFor="required-chk" className="text-sm">Bắt buộc</label>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isPending}>Huỷ</Button>
          <Button onClick={submit} disabled={isPending}>
            {isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : (isEdit ? 'Lưu' : 'Thêm nhóm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---------- Create / Edit option dialog ----------

function OptionDialog({
  open, ogId, option, onClose, mutations,
}: {
  open: boolean;
  ogId: string;
  option?: Option;
  onClose: () => void;
  mutations: ReturnType<typeof useOptionGroupMutations>;
}) {
  const isEdit = !!option;
  const [name, setName] = useState(option?.Name ?? '');
  const [extraPrice, setExtraPrice] = useState(String(option?.ExtraPrice ?? 0));

  const isPending = mutations.createOption.isPending || mutations.updateOption.isPending;

  const submit = () => {
    const price = parseInt(extraPrice, 10);
    if (!name.trim() || isNaN(price) || price < 0) {
      toast.error('Tên và giá thêm không hợp lệ.');
      return;
    }

    if (isEdit && option) {
      mutations.updateOption.mutate(
        { ogId, optId: option.ID, body: { name: name.trim(), extra_price: price } },
        {
          onSuccess: () => { toast.success('Đã cập nhật tuỳ chọn.'); onClose(); },
          onError: (e) => toast.error(getApiErrorMessage(e, 'Cập nhật thất bại')),
        },
      );
    } else {
      mutations.createOption.mutate(
        { ogId, body: { name: name.trim(), extra_price: price } },
        {
          onSuccess: () => { toast.success('Đã thêm tuỳ chọn.'); onClose(); },
          onError: (e) => toast.error(getApiErrorMessage(e, 'Thêm thất bại')),
        },
      );
    }
  };

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Chỉnh sửa tuỳ chọn' : 'Thêm tuỳ chọn'}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <Field label="Tên tuỳ chọn" value={name} onChange={setName} placeholder="VD: Thêm trứng" />
          <Field label="Giá thêm (VNĐ)" value={extraPrice} onChange={setExtraPrice} type="number" placeholder="0" />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isPending}>Huỷ</Button>
          <Button onClick={submit} disabled={isPending}>
            {isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : (isEdit ? 'Lưu' : 'Thêm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---------- Shared inline field ----------

function Field({
  label, value, onChange, placeholder, type = 'text',
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  type?: string;
}) {
  return (
    <div>
      <label className="text-foreground mb-1.5 block text-sm font-medium">{label}</label>
      <Input value={value} type={type} onChange={(e) => onChange(e.target.value)} placeholder={placeholder} />
    </div>
  );
}
