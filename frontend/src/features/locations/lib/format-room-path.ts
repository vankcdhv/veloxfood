// Shared room-path formatting utilities used across features.
// Centralised here so changes to the display convention are made once.

import type { Building, CustomerLocation, Floor, ResolvedRoom } from '../types/location';

/**
 * Formats a saved CustomerLocation's human-readable path.
 * Returns "Toà A · Tầng 1 · P101 · Phòng 101" when all names are present,
 * or falls back to the raw RoomID if the service returned no names.
 */
export function formatRoomPath(loc: CustomerLocation): string {
  const parts = [
    loc.BuildingName,
    loc.FloorName,
    loc.RoomName ? `${loc.RoomCode} · ${loc.RoomName}` : loc.RoomCode,
  ].filter(Boolean);
  return parts.length ? parts.join(' · ') : loc.RoomID;
}

/**
 * Formats a ResolvedRoom (from GET /locations/rooms/:id) the same way.
 * Falls back to the raw RoomID while data is still loading.
 */
export function formatResolvedRoom(r: ResolvedRoom): string {
  const parts = [
    r.BuildingName,
    r.FloorName,
    r.RoomName ? `${r.RoomCode} · ${r.RoomName}` : r.RoomCode,
  ].filter(Boolean);
  return parts.length ? parts.join(' · ') : r.RoomID;
}

/**
 * Formats a Building-level delivery label.
 * Used when the customer stops at Toà level without picking a floor.
 */
export function formatBuildingLabel(building: Building): string {
  return building.Name || building.ID;
}

/**
 * Formats a Floor-level delivery label showing "BuildingName · FloorName".
 * The building name comes from the cached browse list, floor name from browse floors.
 */
export function formatFloorLabel(floor: Floor, buildingName: string): string {
  const parts = [buildingName, floor.Name].filter(Boolean);
  return parts.length ? parts.join(' · ') : floor.ID;
}
