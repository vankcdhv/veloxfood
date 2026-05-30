import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';
import type { Building, CustomerLocation, Floor, ResolvedRoom, Room } from '../types/location';

const ADMIN = `${API_PREFIX}/admin`;
const BROWSE = `${API_PREFIX}/locations`;
const ME = `${API_PREFIX}/me/locations`;

function unwrap<T>(res: { data: ApiResponse<T> }): T {
  if (res.data.data === undefined || res.data.data === null) {
    throw new Error(res.data.error ?? 'Empty response');
  }
  return res.data.data;
}

// ---- Admin: tree management ----
export const adminLocationApi = {
  listBuildings: async () => unwrap(await http.get<ApiResponse<Building[]>>(`${ADMIN}/buildings`)),
  createBuilding: async (name: string, address: string) =>
    unwrap(await http.post<ApiResponse<Building>>(`${ADMIN}/buildings`, { name, address })),
  updateBuilding: async (id: string, body: Partial<{ name: string; address: string; is_active: boolean }>) =>
    unwrap(await http.patch<ApiResponse<Building>>(`${ADMIN}/buildings/${id}`, body)),
  deleteBuilding: async (id: string) => { await http.delete<ApiResponse>(`${ADMIN}/buildings/${id}`); },

  listFloors: async (buildingId: string) =>
    unwrap(await http.get<ApiResponse<Floor[]>>(`${ADMIN}/buildings/${buildingId}/floors`)),
  createFloor: async (buildingId: string, name: string, sortOrder: number) =>
    unwrap(await http.post<ApiResponse<Floor>>(`${ADMIN}/buildings/${buildingId}/floors`, { name, sort_order: sortOrder })),
  deleteFloor: async (id: string) => { await http.delete<ApiResponse>(`${ADMIN}/floors/${id}`); },

  listRooms: async (floorId: string) =>
    unwrap(await http.get<ApiResponse<Room[]>>(`${ADMIN}/floors/${floorId}/rooms`)),
  createRoom: async (floorId: string, code: string, name: string) =>
    unwrap(await http.post<ApiResponse<Room>>(`${ADMIN}/floors/${floorId}/rooms`, { code, name })),
  deleteRoom: async (id: string) => { await http.delete<ApiResponse>(`${ADMIN}/rooms/${id}`); },
};

// ---- Customer: browse (active only) ----
export const browseLocationApi = {
  buildings: async () => unwrap(await http.get<ApiResponse<Building[]>>(`${BROWSE}/buildings`)),
  floors: async (buildingId: string) =>
    unwrap(await http.get<ApiResponse<Floor[]>>(`${BROWSE}/buildings/${buildingId}/floors`)),
  rooms: async (floorId: string) =>
    unwrap(await http.get<ApiResponse<Room[]>>(`${BROWSE}/floors/${floorId}/rooms`)),
  // Resolve a single room to its full building→floor→room path.
  resolveRoom: async (roomId: string) =>
    unwrap(await http.get<ApiResponse<ResolvedRoom>>(`${BROWSE}/rooms/${roomId}`)),
};

// ---- Customer: saved locations ----
export const myLocationApi = {
  list: async () => unwrap(await http.get<ApiResponse<CustomerLocation[]>>(ME)),
  add: async (roomId: string, label: string, isDefault: boolean) =>
    unwrap(await http.post<ApiResponse<CustomerLocation>>(ME, { room_id: roomId, label, is_default: isDefault })),
  remove: async (id: string) => { await http.delete<ApiResponse>(`${ME}/${id}`); },
  setDefault: async (id: string) => { await http.patch<ApiResponse>(`${ME}/${id}/default`, {}); },
};
