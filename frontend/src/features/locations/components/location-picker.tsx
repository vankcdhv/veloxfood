'use client';

// Cascading Building→Floor→Room picker backed by the browse hooks.
// Manages its own internal cascade state; resets child selects when a parent changes.
// Use level='room' for a full 3-level pick (yields roomId via onSelect).
// Use level='building' for building-only pick (yields buildingId via onSelect).
//
// To fully reset this picker from a parent, pass a React key that changes (e.g. a form
// submission counter). Internal building/floor state is intentionally kept on cascade
// changes so the user doesn't lose context while browsing.

import { useState } from 'react';
import { cn } from '@/shared/lib/utils';
import { useBrowseBuildings, useBrowseFloors, useBrowseRooms } from '../hooks/use-locations';

interface LocationPickerProps {
  /** 'room' = all 3 selects, yields roomId. 'building' = building select only, yields buildingId. */
  level?: 'building' | 'room';
  /** Controlled value for the final selection (roomId or buildingId). Pass null/'' to reset. */
  value?: string;
  /** Called with the selected id (roomId or buildingId) when the deepest relevant select changes. */
  onSelect: (id: string) => void;
  className?: string;
}

export function LocationPicker({ level = 'room', value, onSelect, className }: LocationPickerProps) {
  const [buildingId, setBuildingId] = useState('');
  const [floorId, setFloorId] = useState('');

  const buildings = useBrowseBuildings();
  const floors = useBrowseFloors(level === 'room' ? buildingId || null : null);
  const rooms = useBrowseRooms(level === 'room' ? floorId || null : null);

  const handleBuildingChange = (id: string) => {
    setBuildingId(id);
    setFloorId('');
    if (level === 'building') {
      onSelect(id);
    } else {
      // Room-level: building changed, downstream not yet selected.
      onSelect('');
    }
  };

  const handleFloorChange = (id: string) => {
    setFloorId(id);
    // Room not yet selected — clear upstream selection.
    onSelect('');
  };

  const handleRoomChange = (id: string) => {
    onSelect(id);
  };

  return (
    <div className={cn('space-y-2', className)}>
      <SelectField
        label="Toà nhà"
        value={buildingId}
        onChange={handleBuildingChange}
        options={(buildings.data ?? []).map((b) => ({ value: b.ID, label: b.Name }))}
        loading={buildings.isLoading}
      />

      {level === 'room' && (
        <>
          <SelectField
            label="Tầng"
            value={floorId}
            onChange={handleFloorChange}
            options={(floors.data ?? []).map((f) => ({ value: f.ID, label: f.Name }))}
            disabled={!buildingId}
            loading={floors.isFetching}
          />
          <SelectField
            label="Phòng"
            value={value ?? ''}
            onChange={handleRoomChange}
            options={(rooms.data ?? []).map((r) => ({
              value: r.ID,
              label: r.Code + (r.Name ? ` · ${r.Name}` : ''),
            }))}
            disabled={!floorId}
            loading={rooms.isFetching}
          />
        </>
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
