'use client';

import { useRef, useState } from 'react';
import { ImagePlus, Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useVendorStoreMutations } from '../hooks/use-stores';
import type { Category, MenuItem } from '../types/store';

// ---------- Inline image upload trigger ----------

export function ImageUploadButton({ storeId, itemId }: { storeId: string; itemId: string }) {
  const m = useVendorStoreMutations(storeId);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (!file.type.startsWith('image/')) {
      toast.error('Vui lòng chọn file ảnh (jpg, png, webp…).');
      return;
    }
    m.uploadMenuItemImage.mutate(
      { itemId, file },
      {
        onSuccess: () => toast.success('Đã cập nhật ảnh món.'),
        onError: (err) => toast.error(getApiErrorMessage(err, 'Tải ảnh thất bại')),
      },
    );
    e.target.value = '';
  };

  return (
    <>
      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        className="hidden"
        onChange={handleFile}
      />
      <button
        type="button"
        aria-label="Tải ảnh lên"
        onClick={() => inputRef.current?.click()}
        disabled={m.uploadMenuItemImage.isPending}
        className="text-muted-foreground hover:text-primary shrink-0 p-1 opacity-0 group-hover:opacity-100 transition-opacity disabled:opacity-30"
      >
        {m.uploadMenuItemImage.isPending
          ? <Loader2 className="h-4 w-4 animate-spin" />
          : <ImagePlus className="h-4 w-4" />}
      </button>
    </>
  );
}

// ---------- Add menu item dialog ----------

export function AddMenuItemDialog({
  storeId, category, onClose,
}: {
  storeId: string;
  category: Category | null;
  onClose: () => void;
}) {
  const m = useVendorStoreMutations(storeId);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [price, setPrice] = useState('');

  const submit = () => {
    if (!category) return;
    const p = parseInt(price, 10);
    if (!name.trim() || isNaN(p) || p < 0) {
      toast.error('Vui lòng nhập tên và giá hợp lệ.');
      return;
    }
    m.createMenuItem.mutate(
      { category_id: category.ID, name: name.trim(), description: description.trim(), price: p },
      {
        onSuccess: () => {
          toast.success('Đã thêm món.');
          setName(''); setDescription(''); setPrice('');
          onClose();
        },
        onError: (e) => toast.error(getApiErrorMessage(e, 'Thêm món thất bại')),
      },
    );
  };

  return (
    <Dialog open={!!category} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Thêm món — {category?.Name}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <Field label="Tên món" value={name} onChange={setName} placeholder="VD: Cơm sườn" />
          <Field label="Mô tả" value={description} onChange={setDescription} placeholder="Tuỳ chọn" />
          <Field label="Giá (VNĐ)" value={price} onChange={setPrice} placeholder="VD: 35000" type="number" />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={m.createMenuItem.isPending}>Huỷ</Button>
          <Button onClick={submit} disabled={m.createMenuItem.isPending}>
            {m.createMenuItem.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Thêm món'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---------- Edit menu item dialog ----------
// Parent must remount via key={item?.ID} to reset state when editing a different item.

export function EditMenuItemDialog({
  storeId, item, onClose,
}: {
  storeId: string;
  item: MenuItem | null;
  onClose: () => void;
}) {
  const m = useVendorStoreMutations(storeId);
  // Initial values come from item at mount; caller remounts via key={item?.ID}.
  const [name, setName] = useState(item?.Name ?? '');
  const [description, setDescription] = useState(item?.Description ?? '');
  const [price, setPrice] = useState(item ? String(item.Price) : '');
  const [tags, setTags] = useState(item?.Tags ?? '');
  const [imageURL, setImageURL] = useState(item?.ImageURL ?? '');

  const submit = () => {
    if (!item) return;
    const p = parseInt(price, 10);
    if (!name.trim() || isNaN(p) || p < 0) {
      toast.error('Vui lòng nhập tên và giá hợp lệ.');
      return;
    }
    const url = imageURL.trim();
    if (url && !/^https?:\/\//i.test(url)) {
      toast.error('URL ảnh phải bắt đầu bằng http:// hoặc https://');
      return;
    }
    m.updateMenuItem.mutate(
      {
        itemId: item.ID,
        body: { name: name.trim(), description: description.trim(), price: p, tags: tags.trim(), image_url: imageURL.trim() },
      },
      {
        onSuccess: () => {
          toast.success('Đã cập nhật món.');
          onClose();
        },
        onError: (e) => toast.error(getApiErrorMessage(e, 'Cập nhật thất bại')),
      },
    );
  };

  return (
    <Dialog open={!!item} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Chỉnh sửa món</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <Field label="Tên món" value={name} onChange={setName} placeholder="VD: Cơm sườn" />
          <Field label="Mô tả" value={description} onChange={setDescription} placeholder="Tuỳ chọn" />
          <Field label="Giá (VNĐ)" value={price} onChange={setPrice} placeholder="VD: 35000" type="number" />
          <Field
            label="Nhãn (cách nhau bởi dấu phẩy)"
            value={tags}
            onChange={setTags}
            placeholder="VD: bestseller,new"
          />
          <div>
            <Field
              label="Ảnh món (URL)"
              value={imageURL}
              onChange={setImageURL}
              placeholder="https://… (hoặc dùng nút tải ảnh)"
            />
            {/^https?:\/\//i.test(imageURL.trim()) && (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={imageURL.trim()}
                alt="Xem trước ảnh món"
                className="border-border mt-2 h-20 w-20 rounded-md border object-cover"
                onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }}
              />
            )}
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={m.updateMenuItem.isPending}>Huỷ</Button>
          <Button onClick={submit} disabled={m.updateMenuItem.isPending}>
            {m.updateMenuItem.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Lưu'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---------- Shared inline field ----------

export function Field({
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
