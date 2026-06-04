import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { adminLocationApi, browseLocationApi, myLocationApi } from '../api/location-api';

export const locationKeys = {
  all: ['locations'] as const,
  adminBuildings: () => [...locationKeys.all, 'admin', 'buildings'] as const,
  adminFloors: (b: string) => [...locationKeys.all, 'admin', 'floors', b] as const,
  adminRooms: (f: string) => [...locationKeys.all, 'admin', 'rooms', f] as const,
  browseBuildings: () => [...locationKeys.all, 'browse', 'buildings'] as const,
  browseFloors: (b: string) => [...locationKeys.all, 'browse', 'floors', b] as const,
  browseRooms: (f: string) => [...locationKeys.all, 'browse', 'rooms', f] as const,
  resolveRoom: (id: string) => [...locationKeys.all, 'resolve', 'room', id] as const,
  mine: () => [...locationKeys.all, 'mine'] as const,
};

// ---- Admin ----
export const useAdminBuildings = () =>
  useQuery({ queryKey: locationKeys.adminBuildings(), queryFn: adminLocationApi.listBuildings });

export const useAdminFloors = (buildingId: string | null) =>
  useQuery({
    queryKey: locationKeys.adminFloors(buildingId ?? ''),
    queryFn: () => adminLocationApi.listFloors(buildingId!),
    enabled: !!buildingId,
  });

export const useAdminRooms = (floorId: string | null) =>
  useQuery({
    queryKey: locationKeys.adminRooms(floorId ?? ''),
    queryFn: () => adminLocationApi.listRooms(floorId!),
    enabled: !!floorId,
  });

export function useLocationMutations() {
  const qc = useQueryClient();
  const invalidate = () => qc.invalidateQueries({ queryKey: locationKeys.all });
  return {
    createBuilding: useMutation({ mutationFn: (v: { name: string; address: string }) => adminLocationApi.createBuilding(v.name, v.address), onSuccess: invalidate }),
    deleteBuilding: useMutation({ mutationFn: (id: string) => adminLocationApi.deleteBuilding(id), onSuccess: invalidate }),
    createFloor: useMutation({ mutationFn: (v: { buildingId: string; name: string; sortOrder: number }) => adminLocationApi.createFloor(v.buildingId, v.name, v.sortOrder), onSuccess: invalidate }),
    deleteFloor: useMutation({ mutationFn: (id: string) => adminLocationApi.deleteFloor(id), onSuccess: invalidate }),
    createRoom: useMutation({ mutationFn: (v: { floorId: string; code: string; name: string }) => adminLocationApi.createRoom(v.floorId, v.code, v.name), onSuccess: invalidate }),
    deleteRoom: useMutation({ mutationFn: (id: string) => adminLocationApi.deleteRoom(id), onSuccess: invalidate }),
  };
}

// ---- Customer browse ----
export const useBrowseBuildings = () =>
  useQuery({ queryKey: locationKeys.browseBuildings(), queryFn: browseLocationApi.buildings });
export const useBrowseFloors = (buildingId: string | null) =>
  useQuery({ queryKey: locationKeys.browseFloors(buildingId ?? ''), queryFn: () => browseLocationApi.floors(buildingId!), enabled: !!buildingId });
export const useBrowseRooms = (floorId: string | null) =>
  useQuery({ queryKey: locationKeys.browseRooms(floorId ?? ''), queryFn: () => browseLocationApi.rooms(floorId!), enabled: !!floorId });

// Resolves a roomId to its full building→floor→room path. Cached per roomId.
export const useResolveRoom = (roomId: string | null) =>
  useQuery({
    queryKey: locationKeys.resolveRoom(roomId ?? ''),
    queryFn: () => browseLocationApi.resolveRoom(roomId!),
    enabled: !!roomId,
    // Room paths rarely change; keep in cache for the session.
    staleTime: 5 * 60 * 1_000,
  });

// ---- Customer saved locations ----
export const useMyLocations = () =>
  useQuery({ queryKey: locationKeys.mine(), queryFn: myLocationApi.list });

export function useMyLocationMutations() {
  const qc = useQueryClient();
  const invalidate = () => qc.invalidateQueries({ queryKey: locationKeys.mine() });
  return {
    add: useMutation({ mutationFn: (v: { locationId: string; level: string; label: string; isDefault: boolean }) => myLocationApi.add(v.locationId, v.level, v.label, v.isDefault), onSuccess: invalidate }),
    remove: useMutation({ mutationFn: (id: string) => myLocationApi.remove(id), onSuccess: invalidate }),
    setDefault: useMutation({ mutationFn: (id: string) => myLocationApi.setDefault(id), onSuccess: invalidate }),
  };
}
