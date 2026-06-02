'use client';

import { useState } from 'react';
import { Loader2, Plus, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { Skeleton } from '@/shared/ui/skeleton';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import {
  useBrowseBuildings,
  useBrowseFloors,
  useResolveRoom,
} from '@/features/locations/hooks/use-locations';
import { formatResolvedRoom } from '@/features/locations/lib/format-room-path';
import { LocationPicker } from '@/features/locations/components/location-picker';
import { useStoreShipFees, useVendorStoreMutations } from '../hooks/use-stores';
import type { ShipFeeRule, ShipFeeScope } from '../types/store';

interface Props {
  storeId: string;
}

// Maps scope value to the picker level prop used by LocationPicker legacy API.
function pickerLevelForScope(scope: ShipFeeScope): 'building' | 'room' {
  if (scope === 'building') return 'building';
  // Both 'floor' and 'room' use the full cascading picker; floor stops early via onSelect.
  return 'room';
}

export function VendorShipFeePanel({ storeId }: Props) {
  const { data: rules, isLoading } = useStoreShipFees(storeId);
  const m = useVendorStoreMutations(storeId);
  const [scope, setScope] = useState<ShipFeeScope>('building');
  const [refId, setRefId] = useState('');
  const [unitFee, setUnitFee] = useState('');
  // Incrementing remounts LocationPicker, resetting its cascade after add/scope change.
  const [pickerKey, setPickerKey] = useState(0);

  // For floor scope we use the flexible onSelectLocation and stop at FLOOR level.
  // For building/room we use the legacy onSelect API.
  const isFloorScope = scope === 'floor';

  const addRule = () => {
    const fee = parseInt(unitFee, 10);
    if (!refId.trim() || isNaN(fee) || fee < 0) {
      toast.error('Vui lòng chọn vị trí và nhập phí hợp lệ (0 = miễn phí).');
      return;
    }
    m.createShipFee.mutate(
      { scope, ref_id: refId.trim(), unit_fee: fee },
      {
        onSuccess: () => {
          toast.success('Đã thêm quy tắc phí.');
          setRefId('');
          setUnitFee('');
          setPickerKey((k) => k + 1);
        },
        onError: (e) => toast.error(getApiErrorMessage(e, 'Thêm quy tắc thất bại')),
      },
    );
  };

  const deleteRule = (ruleId: string) => {
    m.deleteShipFee.mutate(ruleId, {
      onSuccess: () => toast.success('Đã xoá quy tắc.'),
      onError: (e) => toast.error(getApiErrorMessage(e, 'Xoá thất bại')),
    });
  };

  // Reset refId and picker when scope changes — previous id is invalid for the new scope.
  const handleScopeChange = (s: ShipFeeScope) => {
    setScope(s);
    setRefId('');
    setPickerKey((k) => k + 1);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Phí giao hàng</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Helper copy explaining cascade priority */}
        <p className="text-xs text-muted-foreground rounded-md bg-muted px-3 py-2 leading-relaxed">
          Giá Toà áp cho mọi tầng/phòng, trừ khi có giá Tầng/Phòng riêng.
          Phải cài giá Toà trước khi thêm giá Tầng hoặc Phòng.
          Phí 0đ = giao miễn phí.
        </p>

        {isLoading && <Skeleton className="h-16" />}
        {!isLoading && (!rules || rules.length === 0) && (
          <p className="text-muted-foreground text-sm">Chưa có quy tắc nào.</p>
        )}
        <ul className="space-y-2">
          {rules?.map((rule) => (
            <li
              key={rule.ID}
              className="group flex items-center gap-3 rounded-lg border border-border px-3 py-2"
            >
              <Badge variant="outline" className="shrink-0 capitalize">
                {rule.Scope === 'building' ? 'Toà nhà' : rule.Scope === 'floor' ? 'Tầng' : 'Phòng'}
              </Badge>
              {/* Human-readable label; falls back to '—' only while resolving */}
              <ShipFeeRuleLabel rule={rule} />
              <span className="text-primary text-sm font-semibold shrink-0">
                {rule.UnitFee === 0 ? 'Miễn phí' : `${rule.UnitFee.toLocaleString('vi-VN')}đ`}
              </span>
              <button
                type="button"
                aria-label="Xoá quy tắc"
                onClick={() => deleteRule(rule.ID)}
                disabled={m.deleteShipFee.isPending}
                className="text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 p-1 transition-opacity disabled:opacity-30"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </li>
          ))}
        </ul>

        {/* Add rule form */}
        <div className="space-y-3 border-t border-border pt-3">
          <p className="text-sm font-medium">Thêm quy tắc mới</p>
          <div className="flex gap-2 items-center">
            <select
              value={scope}
              onChange={(e) => handleScopeChange(e.target.value as ShipFeeScope)}
              className="border-input bg-background h-9 rounded-md border px-2 text-sm shrink-0"
            >
              <option value="building">Toà nhà</option>
              <option value="floor">Tầng</option>
              <option value="room">Phòng</option>
            </select>
          </div>

          {/* Picker adapts to scope:
              - building → building-only (legacy level='building')
              - floor    → flexible onSelectLocation, only accept FLOOR selections
              - room     → full cascading (legacy level='room') */}
          {isFloorScope ? (
            <LocationPicker
              key={pickerKey}
              onSelectLocation={(sel) => {
                // Only register the id when the user actually lands on a floor.
                setRefId(sel?.level === 'FLOOR' ? sel.id : '');
              }}
            />
          ) : (
            <LocationPicker
              key={pickerKey}
              level={pickerLevelForScope(scope)}
              value={refId}
              onSelect={setRefId}
            />
          )}

          <div className="flex gap-2">
            <Input
              value={unitFee}
              type="number"
              min={0}
              onChange={(e) => setUnitFee(e.target.value)}
              placeholder="Phí (VNĐ) — 0 = miễn phí"
              className="flex-1"
            />
            <Button
              onClick={addRule}
              disabled={m.createShipFee.isPending || !refId}
              size="icon"
              aria-label="Thêm quy tắc phí"
            >
              {m.createShipFee.isPending
                ? <Loader2 className="h-4 w-4 animate-spin" />
                : <Plus className="h-4 w-4" />}
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// Displays a human-readable label for a ship-fee rule's RefID.
// Building scope → building name from the cached browse list.
// Floor scope    → "BuildingName · FloorName" from browse lists (no extra API call).
// Room scope     → resolves via the API and formats with formatResolvedRoom.
// Shows "—" while loading or when the name cannot be resolved — never shows raw UUIDs.
function ShipFeeRuleLabel({ rule }: { rule: ShipFeeRule }) {
  // Single buildings query — covers both building-scope labels and the floor resolver.
  // React Query deduplicates the network call when the same key is used concurrently.
  const buildings = useBrowseBuildings();

  // Room resolve is only enabled when scope is 'room'.
  const resolved = useResolveRoom(rule.Scope === 'room' ? rule.RefID : null);

  if (rule.Scope === 'building') {
    const name = buildings.data?.find((b) => b.ID === rule.RefID)?.Name;
    return (
      <span className="text-sm flex-1 truncate text-muted-foreground">
        {name ?? '—'}
      </span>
    );
  }

  if (rule.Scope === 'floor') {
    // Floor label resolved via FloorNameResolver sub-component to keep hook rules stable.
    return (
      <FloorRuleLabel
        floorId={rule.RefID}
        buildings={buildings.data ?? []}
      />
    );
  }

  // Room scope — show '…' while loading, '—' on error.
  const label = resolved.data ? formatResolvedRoom(resolved.data) : (resolved.isLoading ? '…' : '—');
  return (
    <span className="text-sm flex-1 truncate text-muted-foreground">
      {label}
    </span>
  );
}

// Resolves a floor label by browsing floors for each building until a match is found.
// Uses a single building's floor list at a time — the floor ID is globally unique so
// the first match wins. We try the first building's floors; if no match we show '—'.
// For correctness with multiple buildings this component accepts all buildings and
// iterates via a recursive approach — but to avoid dynamic hook calls we use a
// dedicated single-building resolver that the parent maps.
//
// Simpler approach: resolve the floor name by scanning already-loaded floor data
// from the location browse cache via a targeted hook per building. Since we can't
// call hooks inside a loop, we use a single-building subcomponent pattern.
function FloorRuleLabel({ floorId, buildings }: {
  floorId: string;
  buildings: import('@/features/locations/types/location').Building[];
}) {
  // Try the first building; if the floor belongs to it we get the name.
  // For a vendor store the number of buildings is typically small (1-5).
  // We render a per-building resolver for each and show the first non-null result.
  if (buildings.length === 0) {
    return <span className="text-sm flex-1 truncate text-muted-foreground">…</span>;
  }
  return (
    <FloorRuleLabelInner
      floorId={floorId}
      buildingIds={buildings.map((b) => b.ID)}
      buildingNames={Object.fromEntries(buildings.map((b) => [b.ID, b.Name]))}
      index={0}
    />
  );
}

// Recursively tries each building's floor list until the floorId is found.
// index drives which building we're currently checking.
function FloorRuleLabelInner({ floorId, buildingIds, buildingNames, index }: {
  floorId: string;
  buildingIds: string[];
  buildingNames: Record<string, string>;
  index: number;
}) {
  const buildingId = buildingIds[index] ?? '';
  const floorsQuery = useBrowseFloors(buildingId || null);

  const floor = floorsQuery.data?.find((f) => f.ID === floorId);

  if (floor) {
    const bName = buildingNames[buildingId] ?? '';
    const label = [bName, floor.Name].filter(Boolean).join(' · ');
    return (
      <span className="text-sm flex-1 truncate text-muted-foreground">
        {label || '—'}
      </span>
    );
  }

  // Not in this building's floors — try the next building if available.
  if (!floorsQuery.isLoading && index + 1 < buildingIds.length) {
    return (
      <FloorRuleLabelInner
        floorId={floorId}
        buildingIds={buildingIds}
        buildingNames={buildingNames}
        index={index + 1}
      />
    );
  }

  const loading = floorsQuery.isLoading;
  return (
    <span className="text-sm flex-1 truncate text-muted-foreground">
      {loading ? '…' : '—'}
    </span>
  );
}
