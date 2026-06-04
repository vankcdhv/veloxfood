'use client';

import { useState, type ReactNode } from 'react';
import { ChevronRight, Loader2, Plus, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { Skeleton } from '@/shared/ui/skeleton';
import { ConfirmDialog } from '@/shared/ui/confirm-dialog';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { cn } from '@/shared/lib/utils';
import {
  useAdminBuildings,
  useAdminFloors,
  useAdminRooms,
  useLocationMutations,
} from '../hooks/use-locations';

type DeleteKind = 'building' | 'floor' | 'room';
interface PendingDelete {
  kind: DeleteKind;
  id: string;
  label: string;
}

const DELETE_DESC: Record<DeleteKind, string> = {
  building: 'Xoá toà nhà sẽ xoá toàn bộ tầng và phòng bên trong.',
  floor: 'Xoá tầng sẽ xoá toàn bộ phòng bên trong.',
  room: 'Phòng này sẽ bị xoá khỏi hệ thống.',
};

// Master-detail manager: Buildings → Floors → Rooms (3 selectable columns).
export function LocationTreeManager() {
  const [buildingId, setBuildingId] = useState<string | null>(null);
  const [floorId, setFloorId] = useState<string | null>(null);
  const [pending, setPending] = useState<PendingDelete | null>(null);

  const buildings = useAdminBuildings();
  const floors = useAdminFloors(buildingId);
  const rooms = useAdminRooms(floorId);
  const m = useLocationMutations();

  const err = (e: unknown) => toast.error(getApiErrorMessage(e, 'Thao tác thất bại'));
  const deleting =
    m.deleteBuilding.isPending || m.deleteFloor.isPending || m.deleteRoom.isPending;

  const confirmDelete = () => {
    if (!pending) return;
    const opts = {
      onSuccess: () => { toast.success('Đã xoá.'); setPending(null); },
      onError: (e: unknown) => { err(e); setPending(null); },
    };
    if (pending.kind === 'building') {
      m.deleteBuilding.mutate(pending.id, opts);
      if (buildingId === pending.id) { setBuildingId(null); setFloorId(null); }
    } else if (pending.kind === 'floor') {
      m.deleteFloor.mutate(pending.id, opts);
      if (floorId === pending.id) setFloorId(null);
    } else {
      m.deleteRoom.mutate(pending.id, opts);
    }
  };

  return (
    <>
    <div className="grid gap-4 lg:grid-cols-3">
      {/* Buildings */}
      <Column
        title="Toà nhà"
        loading={buildings.isLoading}
        addPlaceholder="Tên toà nhà mới"
        onAdd={(name) => m.createBuilding.mutate({ name, address: '' }, { onError: err })}
        adding={m.createBuilding.isPending}
      >
        {buildings.data?.map((b) => (
          <Row
            key={b.ID}
            label={b.Name}
            active={buildingId === b.ID}
            onSelect={() => { setBuildingId(b.ID); setFloorId(null); }}
            onDelete={() => setPending({ kind: 'building', id: b.ID, label: b.Name })}
            chevron
          />
        ))}
        {buildings.data?.length === 0 && <Empty>Chưa có toà nhà</Empty>}
      </Column>

      {/* Floors */}
      <Column
        title="Tầng"
        disabled={!buildingId}
        loading={floors.isLoading}
        addPlaceholder="Tên tầng mới"
        onAdd={(name) => buildingId && m.createFloor.mutate({ buildingId, name, sortOrder: 0 }, { onError: err })}
        adding={m.createFloor.isPending}
      >
        {!buildingId && <Empty>Chọn một toà nhà</Empty>}
        {buildingId && floors.data?.map((f) => (
          <Row
            key={f.ID}
            label={f.Name}
            active={floorId === f.ID}
            onSelect={() => setFloorId(f.ID)}
            onDelete={() => setPending({ kind: 'floor', id: f.ID, label: f.Name })}
            chevron
          />
        ))}
        {buildingId && floors.data?.length === 0 && <Empty>Chưa có tầng</Empty>}
      </Column>

      {/* Rooms */}
      <Column
        title="Phòng"
        disabled={!floorId}
        loading={rooms.isLoading}
        addPlaceholder="Mã phòng mới (vd P101)"
        onAdd={(code) => floorId && m.createRoom.mutate({ floorId, code, name: '' }, { onError: err })}
        adding={m.createRoom.isPending}
      >
        {!floorId && <Empty>Chọn một tầng</Empty>}
        {floorId && rooms.data?.map((r) => (
          <Row
            key={r.ID}
            label={r.Code + (r.Name ? ` · ${r.Name}` : '')}
            onDelete={() => setPending({ kind: 'room', id: r.ID, label: r.Code })}
          />
        ))}
        {floorId && rooms.data?.length === 0 && <Empty>Chưa có phòng</Empty>}
      </Column>
    </div>

    <ConfirmDialog
      open={!!pending}
      onOpenChange={(o) => !o && setPending(null)}
      title={`Xoá "${pending?.label ?? ''}"?`}
      description={pending ? DELETE_DESC[pending.kind] : ''}
      confirmLabel="Xoá"
      destructive
      loading={deleting}
      onConfirm={confirmDelete}
    />
    </>
  );
}

function Column({
  title, children, onAdd, addPlaceholder, adding, loading, disabled,
}: {
  title: string;
  children: ReactNode;
  onAdd: (value: string) => void;
  addPlaceholder: string;
  adding?: boolean;
  loading?: boolean;
  disabled?: boolean;
}) {
  const [value, setValue] = useState('');
  const submit = () => {
    const v = value.trim();
    if (!v) return;
    onAdd(v);
    setValue('');
  };
  return (
    <section className="border-border bg-card flex flex-col rounded-lg border">
      <header className="border-border border-b px-4 py-3">
        <h2 className="font-serif text-base font-semibold">{title}</h2>
      </header>
      <div className="flex-1 space-y-1 p-2 min-h-40">
        {loading ? <Skeleton className="m-2 h-8" /> : children}
      </div>
      <div className="border-border flex gap-2 border-t p-2">
        <Input
          value={value}
          disabled={disabled || adding}
          placeholder={addPlaceholder}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && submit()}
        />
        <Button size="icon" onClick={submit} disabled={disabled || adding} aria-label={`Thêm ${title}`}>
          {adding ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
        </Button>
      </div>
    </section>
  );
}

function Row({
  label, active, onSelect, onDelete, chevron,
}: {
  label: string;
  active?: boolean;
  onSelect?: () => void;
  onDelete: () => void;
  chevron?: boolean;
}) {
  return (
    <div
      className={cn(
        'group flex items-center gap-1 rounded-md px-2 py-1.5 text-sm transition-colors',
        active ? 'bg-primary/10 text-primary' : 'hover:bg-accent/50',
      )}
    >
      <button
        type="button"
        onClick={onSelect}
        disabled={!onSelect}
        className="flex flex-1 items-center gap-1 text-left disabled:cursor-default"
      >
        <span className="flex-1 truncate">{label}</span>
        {chevron && <ChevronRight className="text-muted-foreground h-4 w-4 shrink-0" />}
      </button>
      <button
        type="button"
        onClick={onDelete}
        aria-label="Xoá"
        className="text-muted-foreground hover:text-destructive shrink-0 rounded p-1 opacity-60 transition-opacity group-hover:opacity-100"
      >
        <Trash2 className="h-4 w-4" />
      </button>
    </div>
  );
}

function Empty({ children }: { children: ReactNode }) {
  return <p className="text-muted-foreground px-3 py-6 text-center text-sm">{children}</p>;
}
