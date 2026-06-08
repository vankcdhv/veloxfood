import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';
import type {
  Category,
  Combo,
  ComboItem,
  HoursChangeRequest,
  MenuItem,
  MenuItemSearchResult,
  Option,
  OptionGroup,
  ShipFeeResult,
  ShipFeeRule,
  Store,
  StoreHoursResponse,
} from '../types/store';

const STORES = `${API_PREFIX}/stores`;
const ADMIN = `${API_PREFIX}/admin/stores`;

function unwrap<T>(res: { data: ApiResponse<T> }): T {
  if (res.data.data === undefined || res.data.data === null) {
    throw new Error(res.data.error ?? 'Empty response');
  }
  return res.data.data;
}

const MENU_ITEMS = `${API_PREFIX}/menu-items`;

// ---- Global search ----
export const searchApi = {
  // Returns best-match dish results across all active stores. Blank q → empty [].
  menuItems: async (q: string, limit = 24): Promise<MenuItemSearchResult[]> => {
    if (!q.trim()) return [];
    const res = await http.get<ApiResponse<MenuItemSearchResult[]>>(`${MENU_ITEMS}/search`, {
      params: { q, limit },
    });
    if (res.data.data === undefined || res.data.data === null) return [];
    return res.data.data;
  },
};

// ---- Public browse ----
export const browseStoreApi = {
  list: async () => unwrap(await http.get<ApiResponse<Store[]>>(STORES)),
  // Returns only the stores owned by the authenticated user (vendor console).
  mine: async () => unwrap(await http.get<ApiResponse<Store[]>>(`${STORES}/mine`)),
  get: async (id: string) => unwrap(await http.get<ApiResponse<Store>>(`${STORES}/${id}`)),
  // The backend returns a flat list of "on" items; the UI groups them by category.
  menu: async (id: string) => unwrap(await http.get<ApiResponse<MenuItem[]>>(`${STORES}/${id}/menu`)),
  categories: async (id: string) =>
    unwrap(await http.get<ApiResponse<Category[]>>(`${STORES}/${id}/categories`)),
  shipFee: async (id: string, level: string, locationId: string) =>
    unwrap(await http.get<ApiResponse<ShipFeeResult>>(`${STORES}/${id}/ship-fee`, { params: { level, location_id: locationId } })),
  hours: async (id: string) =>
    unwrap(await http.get<ApiResponse<StoreHoursResponse>>(`${STORES}/${id}/hours`)),
};

// ---- Vendor (store owner) ----
export const vendorStoreApi = {
  updateProfile: async (
    id: string,
    body: Partial<{ name: string; business_type: string; address: string; phone: string; prep_minutes: number }>,
  ) => unwrap(await http.patch<ApiResponse<Store>>(`${STORES}/${id}`, body)),

  setSaleStatus: async (id: string, saleStatus: string) =>
    unwrap(await http.patch<ApiResponse<Store>>(`${STORES}/${id}/sale-status`, { sale_status: saleStatus })),

  setPickup: async (id: string, pickupEnabled: boolean) =>
    unwrap(await http.patch<ApiResponse<Store>>(`${STORES}/${id}/pickup`, { enabled: pickupEnabled })),

  // Categories
  createCategory: async (storeId: string, name: string, sortOrder: number) =>
    unwrap(await http.post<ApiResponse<Category>>(`${STORES}/${storeId}/categories`, { name, sort_order: sortOrder })),
  updateCategory: async (storeId: string, catId: string, body: Partial<{ name: string; sort_order: number }>) =>
    unwrap(await http.patch<ApiResponse<Category>>(`${STORES}/${storeId}/categories/${catId}`, body)),
  deleteCategory: async (storeId: string, catId: string) => {
    await http.delete<ApiResponse>(`${STORES}/${storeId}/categories/${catId}`);
  },

  // Menu items
  createMenuItem: async (
    storeId: string,
    body: { category_id: string; name: string; description: string; price: number },
  ) => unwrap(await http.post<ApiResponse<MenuItem>>(`${STORES}/${storeId}/menu`, body)),
  updateMenuItem: async (storeId: string, itemId: string, body: Partial<{ category_id: string; name: string; description: string; price: number; tags: string; image_url: string }>) =>
    unwrap(await http.patch<ApiResponse<MenuItem>>(`${STORES}/${storeId}/menu/${itemId}`, body)),
  setMenuItemStatus: async (storeId: string, itemId: string, status: string) =>
    unwrap(await http.patch<ApiResponse<MenuItem>>(`${STORES}/${storeId}/menu/${itemId}/status`, { status })),
  deleteMenuItem: async (storeId: string, itemId: string) => {
    await http.delete<ApiResponse>(`${STORES}/${storeId}/menu/${itemId}`);
  },

  // Ship-fee rules
  listShipFees: async (storeId: string) =>
    unwrap(await http.get<ApiResponse<ShipFeeRule[]>>(`${STORES}/${storeId}/ship-fees`)),
  createShipFee: async (storeId: string, body: { scope: string; ref_id: string; unit_fee: number }) =>
    unwrap(await http.post<ApiResponse<ShipFeeRule>>(`${STORES}/${storeId}/ship-fees`, body)),
  updateShipFee: async (storeId: string, ruleId: string, body: Partial<{ scope: string; ref_id: string; unit_fee: number }>) =>
    unwrap(await http.patch<ApiResponse<ShipFeeRule>>(`${STORES}/${storeId}/ship-fees/${ruleId}`, body)),
  deleteShipFee: async (storeId: string, ruleId: string) => {
    await http.delete<ApiResponse>(`${STORES}/${storeId}/ship-fees/${ruleId}`);
  },

  // Upload a menu item image; multipart field name is "image". Returns { image_url }.
  uploadMenuItemImage: async (storeId: string, itemId: string, file: File): Promise<string> => {
    const form = new FormData();
    form.append('image', file);
    const res = await http.post<ApiResponse<{ image_url: string }>>(
      `${STORES}/${storeId}/menu/${itemId}/image`,
      form,
      { headers: { 'Content-Type': 'multipart/form-data' } },
    );
    if (!res.data.data?.image_url) throw new Error(res.data.error ?? 'Upload failed');
    return res.data.data.image_url;
  },

  // Hours-change request
  requestHoursChange: async (storeId: string, payload: Record<string, unknown>) =>
    unwrap(await http.post<ApiResponse<HoursChangeRequest>>(`${STORES}/${storeId}/hours-change`, { payload })),

  // Vendor's own hours-change requests with status.
  listHoursChangeRequests: async (storeId: string, status?: string) =>
    unwrap(
      await http.get<ApiResponse<HoursChangeRequest[]>>(
        `${STORES}/${storeId}/hours-change`,
        { params: status ? { status } : {} },
      ),
    ),

  // ---- Option groups ----
  listOptionGroups: async (storeId: string) =>
    unwrap(await http.get<ApiResponse<OptionGroup[]>>(`${STORES}/${storeId}/option-groups`)),
  createOptionGroup: async (storeId: string, body: { name: string; min_select: number; max_select: number; required: boolean }) =>
    unwrap(await http.post<ApiResponse<OptionGroup>>(`${STORES}/${storeId}/option-groups`, body)),
  updateOptionGroup: async (storeId: string, ogId: string, body: Partial<{ name: string; min_select: number; max_select: number; required: boolean }>) =>
    unwrap(await http.patch<ApiResponse<OptionGroup>>(`${STORES}/${storeId}/option-groups/${ogId}`, body)),
  deleteOptionGroup: async (storeId: string, ogId: string) => {
    await http.delete<ApiResponse>(`${STORES}/${storeId}/option-groups/${ogId}`);
  },

  // ---- Options (within a group) ----
  listOptions: async (storeId: string, ogId: string) =>
    unwrap(await http.get<ApiResponse<Option[]>>(`${STORES}/${storeId}/option-groups/${ogId}/options`)),
  createOption: async (storeId: string, ogId: string, body: { name: string; extra_price: number }) =>
    unwrap(await http.post<ApiResponse<Option>>(`${STORES}/${storeId}/option-groups/${ogId}/options`, body)),
  updateOption: async (storeId: string, ogId: string, optId: string, body: Partial<{ name: string; extra_price: number }>) =>
    unwrap(await http.patch<ApiResponse<Option>>(`${STORES}/${storeId}/option-groups/${ogId}/options/${optId}`, body)),
  deleteOption: async (storeId: string, ogId: string, optId: string) => {
    await http.delete<ApiResponse>(`${STORES}/${storeId}/option-groups/${ogId}/options/${optId}`);
  },

  // ---- Attach / detach option groups to a menu item ----
  listMenuItemOptionGroups: async (storeId: string, itemId: string) =>
    unwrap(await http.get<ApiResponse<OptionGroup[]>>(`${STORES}/${storeId}/menu/${itemId}/option-groups`)),
  attachOptionGroup: async (storeId: string, itemId: string, optionGroupId: string) =>
    unwrap(await http.post<ApiResponse<unknown>>(`${STORES}/${storeId}/menu/${itemId}/option-groups`, { option_group_id: optionGroupId })),
  detachOptionGroup: async (storeId: string, itemId: string, ogId: string) => {
    await http.delete<ApiResponse>(`${STORES}/${storeId}/menu/${itemId}/option-groups/${ogId}`);
  },

  // ---- Combos ----
  listCombos: async (storeId: string) =>
    unwrap(await http.get<ApiResponse<Combo[]>>(`${STORES}/${storeId}/combos`)),
  createCombo: async (storeId: string, body: { name: string; price: number }) =>
    unwrap(await http.post<ApiResponse<Combo>>(`${STORES}/${storeId}/combos`, body)),
  updateCombo: async (storeId: string, comboId: string, body: Partial<{ name: string; price: number }>) =>
    unwrap(await http.patch<ApiResponse<Combo>>(`${STORES}/${storeId}/combos/${comboId}`, body)),
  deleteCombo: async (storeId: string, comboId: string) => {
    await http.delete<ApiResponse>(`${STORES}/${storeId}/combos/${comboId}`);
  },

  // ---- Combo items ----
  listComboItems: async (storeId: string, comboId: string) =>
    unwrap(await http.get<ApiResponse<ComboItem[]>>(`${STORES}/${storeId}/combos/${comboId}/items`)),
  addComboItem: async (storeId: string, comboId: string, body: { menu_item_id: string; quantity: number }) =>
    unwrap(await http.post<ApiResponse<ComboItem>>(`${STORES}/${storeId}/combos/${comboId}/items`, body)),
  removeComboItem: async (storeId: string, comboId: string, menuItemId: string) => {
    await http.delete<ApiResponse>(`${STORES}/${storeId}/combos/${comboId}/items/${menuItemId}`);
  },

};

// ---- Admin ----
export const adminStoreApi = {
  list: async () => unwrap(await http.get<ApiResponse<Store[]>>(ADMIN)),
  create: async (body: { vendor_id: string; owner_user_id: string; name: string }) =>
    unwrap(await http.post<ApiResponse<Store>>(ADMIN, body)),
  update: async (id: string, body: Partial<{ name: string; business_type: string; address: string; phone: string }>) =>
    unwrap(await http.patch<ApiResponse<Store>>(`${ADMIN}/${id}`, body)),

  // Hours-change review
  listHoursChangeRequests: async (storeId: string) =>
    unwrap(await http.get<ApiResponse<HoursChangeRequest[]>>(`${ADMIN}/${storeId}/hours-change`)),
  approveHoursChange: async (storeId: string, reqId: string) => {
    await http.post<ApiResponse>(`${ADMIN}/${storeId}/hours-change/${reqId}/approve`, {});
  },
  rejectHoursChange: async (storeId: string, reqId: string) => {
    await http.post<ApiResponse>(`${ADMIN}/${storeId}/hours-change/${reqId}/reject`, {});
  },
};
