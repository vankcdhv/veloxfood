'use client';

// Cascading Building→Floor→Room picker backed by the browse hooks.
// Manages its own internal cascade state; resets child selects when a parent changes.
//
// Legacy API (unchanged): use level + onSelect for callers that only need a single id.
//   level='room'     → all 3 selects, onSelect(roomId) on room change
//   level='building' → building select only, onSelect(buildingId)
//
// Flexible API: use onSelectLocation to receive {level, id} as the user stops at
// any depth (building/floor/room). Selecting a higher level clears deeper state and
// emits the new depth immediately.
//
// To fully reset this picker from a parent, pass a React key that changes (e.g. a
// form submission counter). Internal building/floor state is intentionally kept on
// cascade changes so the user doesn't lose context while browsing.

import { useState } from 'react';
import { cn } from '@/shared/lib/utils';
import { useBrowseBuildings, useBrowseFloors, useBrowseRooms } from '../hooks/use-locations';

export type LocationLevel = 'BUILDING' | 'FLOOR' | 'ROOM';

export interface LocationSelection {
  level: LocationLevel;
  id: string;
}

interface LocationPickerProps {
  /** 'room' = all 3 selects, yields roomId. 'building' = building select only, yields buildingId.
   *  Ignored when onSelectLocation is provided (flexible mode). */
  level?: 'building' | 'room';
  /** Controlled value for the final selection (roomId or buildingId). Pass null/'' to reset.
   *  Only used by the legacy onSelect API. */
  value?: string;
  /** Called with the selected id (roomId or buildingId) when the deepest relevant select changes.
   *  Legacy callback — kept for existing callers. */
  onSelect?: (id: string) => void;
  /** Flexible callback: emitted whenever the user stops at any depth.
   *  Supersedes onSelect/level when provided. Emits null on clear. */
  onSelectLocation?: (sel: LocationSelection | null) => void;
  className?: string;
}

export function LocationPicker({
  level = 'room',
  value,
  onSelect,
  onSelectLocation,
  className,
}: LocationPickerProps) {
  const flexible = !!onSelectLocation;

  const [buildingId, setBuildingId] = useState('');
  const [floorId, setFloorId] = useState('');

  // In flexible mode, show all 3 levels regardless of the `level` prop.
  const showFloor = flexible || level === 'room';
  const showRoom = flexible || level === 'room';

  const buildings = useBrowseBuildings();
  const floors = useBrowseFloors(showFloor ? (buildingId || null) : null);
  const rooms = useBrowseRooms(showRoom ? (floorId || null) : null);

  const handleBuildingChange = (id: string) => {
    setBuildingId(id);
    setFloorId('');
    if (flexible) {
      onSelectLocation?.(id ? { level: 'BUILDING', id } : null);
      return;
    }
    if (level === 'building') {
      onSelect?.(id);
    } else {
      // Room-level legacy: building changed, downstream not yet selected.
      onSelect?.('');
    }
  };

  const handleFloorChange = (id: string) => {
    setFloorId(id);
    if (flexible) {
      onSelectLocation?.(id ? { level: 'FLOOR', id } : (buildingId ? { level: 'BUILDING', id: buildingId } : null));
      return;
    }
    // Legacy: room not yet selected — clear upstream selection.
    onSelect?.('');
  };

  const handleRoomChange = (id: string) => {
    if (flexible) {
      onSelectLocation?.(id ? { level: 'ROOM', id } : (floorId ? { level: 'FLOOR', id: floorId } : (buildingId ? { level: 'BUILDING', id: buildingId } : null)));
      return;
    }
    onSelect?.(id);
  };

  // In legacy mode the room value is controlled externally via `value`.
  // In flexible mode we use uncontrolled (empty string) since the parent tracks via onSelectLocation.
  const roomValue = flexible ? '' : (value ?? '');

  return (
    <div className={cn('space-y-2', className)}>
      <SelectField
        label="Toà nhà"
        value={buildingId}
        onChange={handleBuildingChange}
        options={(buildings.data ?? []).map((b) => ({ value: b.ID, label: b.Name }))}
        loading={buildings.isLoading}
      />

      {showFloor && (
        <SelectField
          label="Tầng"
          value={floorId}
          onChange={handleFloorChange}
          options={(floors.data ?? []).map((f) => ({ value: f.ID, label: f.Name }))}
          disabled={!buildingId}
          loading={floors.isFetching}
        />
      )}

      {showRoom && (
        <SelectField
          label="Phòng"
          value={roomValue}
          onChange={handleRoomChange}
          options={(rooms.data ?? []).map((r) => ({
            value: r.ID,
            label: r.Code + (r.Name ? ` · ${r.Name}` : ''),
          }))}
          disabled={!floorId}
          loading={rooms.isFetching}
        />
      )}
    </div>
  );
}

interface SelectFieldProps {
  label: string;
  value: string;
  onChange: (v: string) => void;
  options: { value: string; label: string }[];
  disabled?: boolean;
  loading?: boolean;
}

function SelectField({ label, value, onChange, options, disabled, loading }: SelectFieldProps) {
  return (
    <div>
      <label className="text-foreground mb-1.5 block text-sm font-medium">{label}</label>
      <select
        value={value}
        disabled={disabled || loading}
        onChange={(e) => onChange(e.target.value)}
        className={cn(
          'border-input bg-background focus-visible:ring-ring h-9 w-full rounded-md border px-3 text-sm focus-visible:ring-2 focus-visible:outline-none disabled:opacity-50',
        )}
      >
        <option value="">{loading ? 'Đang tải…' : '— Chọn —'}</option>
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </div>
  );
}
