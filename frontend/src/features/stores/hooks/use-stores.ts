import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { adminStoreApi, browseStoreApi, searchApi, vendorStoreApi } from '../api/store-api';
import type { MenuCategory, MenuItemSearchFilters } from '../types/store';

const BROWSE_PAGE_SIZE = 12;
const SEARCH_PAGE_SIZE = 12;

export const storeKeys = {
  all: ['stores'] as const,
  menuItemSearch: (q: string, filters: MenuItemSearchFilters = {}) =>
    ['menu-items', 'search', q, filters] as const,
  browseList: () => [...storeKeys.all, 'browse', 'list'] as const,
  browseInfinite: (q: string) => [...storeKeys.all, 'browse', 'infinite', q] as const,
  myList: () => [...storeKeys.all, 'mine'] as const,
  browseDetail: (id: string) => [...storeKeys.all, 'browse', 'detail', id] as const,
  browseMenu: (id: string) => [...storeKeys.all, 'browse', 'menu', id] as const,
  storeHours: (id: string) => [...storeKeys.all, 'hours', id] as const,
  shipFees: (storeId: string) => [...storeKeys.all, 'ship-fees', storeId] as const,
  shipFee: (storeId: string, level: string, locationId: string) => [...storeKeys.all, 'ship-fee', storeId, level, locationId] as const,
  adminList: () => [...storeKeys.all, 'admin', 'list'] as const,
  hoursChange: (storeId: string) => [...storeKeys.all, 'hours-change', storeId] as const,
  vendorHoursChange: (storeId: string) => [...storeKeys.all, 'vendor-hours-change', storeId] as const,
  optionGroups: (storeId: string) => [...storeKeys.all, 'option-groups', storeId] as const,
  options: (storeId: string, ogId: string) => [...storeKeys.all, 'options', storeId, ogId] as const,
  menuItemOptionGroups: (storeId: string, itemId: string) => [...storeKeys.all, 'item-option-groups', storeId, itemId] as const,
  combos: (storeId: string) => [...storeKeys.all, 'combos', storeId] as const,
  comboItems: (storeId: string, comboId: string) => [...storeKeys.all, 'combo-items', storeId, comboId] as const,
};

// ---- Public browse ----
// Full store list (≤100) for store-name lookup maps — NOT the rendered grid.
export const useStores = () =>
  useQuery({ queryKey: storeKeys.browseList(), queryFn: browseStoreApi.list });

// Infinite-scroll page of stores for the browse grid. q filters by name server-side.
export const useBrowseStores = (q: string) =>
  useInfiniteQuery({
    queryKey: storeKeys.browseInfinite(q.trim()),
    queryFn: ({ pageParam }) =>
      browseStoreApi.listPage({ q: q.trim(), limit: BROWSE_PAGE_SIZE, offset: pageParam }),
    initialPageParam: 0,
    getNextPageParam: (lastPage, allPages) => {
      const loaded = allPages.reduce((n, p) => n + p.items.length, 0);
      return loaded < lastPage.total ? loaded : undefined;
    },
  });

// Returns only the stores owned by the authenticated user (vendor console).
export const useMyStores = () =>
  useQuery({ queryKey: storeKeys.myList(), queryFn: browseStoreApi.mine });

export const useStore = (id: string) =>
  useQuery({
    queryKey: storeKeys.browseDetail(id),
    queryFn: () => browseStoreApi.get(id),
    enabled: !!id,
  });

// Fetches the flat menu + categories and groups items into MenuCategory[],
// appending an "Khác" bucket for items with no (or an unknown) category.
export const useStoreMenu = (id: string) =>
  useQuery({
    queryKey: storeKeys.browseMenu(id),
    enabled: !!id,
    queryFn: async (): Promise<MenuCategory[]> => {
      const [items, categories] = await Promise.all([
        browseStoreApi.menu(id),
        browseStoreApi.categories(id),
      ]);
      const groups: MenuCategory[] = categories.map((cat) => ({
        Category: cat,
        Items: items.filter((it) => it.CategoryID === cat.ID),
      }));
      const known = new Set(categories.map((c) => c.ID));
      const orphans = items.filter((it) => !it.CategoryID || !known.has(it.CategoryID));
      if (orphans.length > 0) {
        groups.push({
          Category: { ID: '', StoreID: id, Name: 'Khác', SortOrder: 9999 },
          Items: orphans,
        });
      }
      return groups;
    },
  });

// Search dishes by name across all active stores (infinite scroll). Debounce in the caller.
export const useMenuItemSearch = (q: string, filters: MenuItemSearchFilters = {}) =>
  useInfiniteQuery({
    queryKey: storeKeys.menuItemSearch(q.trim(), filters),
    queryFn: ({ pageParam }) =>
      searchApi.menuItems(q.trim(), { limit: SEARCH_PAGE_SIZE, offset: pageParam }, filters),
    initialPageParam: 0,
    getNextPageParam: (lastPage, allPages) => {
      const loaded = allPages.reduce((n, p) => n + p.items.length, 0);
      return loaded < lastPage.total ? loaded : undefined;
    },
    enabled: q.trim().length >= 1,
  });

// ---- Vendor mutations ----
export function useVendorStoreMutations(storeId: string) {
  const qc = useQueryClient();
  const invalidateStore = () => qc.invalidateQueries({ queryKey: storeKeys.browseDetail(storeId) });
  const invalidateMenu = () => qc.invalidateQueries({ queryKey: storeKeys.browseMenu(storeId) });
  const invalidateShipFees = () => qc.invalidateQueries({ queryKey: storeKeys.shipFees(storeId) });

  const invalidateMine = () => qc.invalidateQueries({ queryKey: storeKeys.myList() });

  return {
    updateProfile: useMutation({
      mutationFn: (body: Parameters<typeof vendorStoreApi.updateProfile>[1]) =>
        vendorStoreApi.updateProfile(storeId, body),
      onSuccess: invalidateStore,
    }),
    uploadAvatar: useMutation({
      mutationFn: (file: File) => vendorStoreApi.uploadStoreAvatar(storeId, file),
      onSuccess: () => { invalidateStore(); invalidateMine(); },
    }),
    setSaleStatus: useMutation({
      mutationFn: (saleStatus: string) => vendorStoreApi.setSaleStatus(storeId, saleStatus),
      onSuccess: invalidateStore,
    }),
    setPickup: useMutation({
      mutationFn: (enabled: boolean) => vendorStoreApi.setPickup(storeId, enabled),
      onSuccess: invalidateStore,
    }),
    createCategory: useMutation({
      mutationFn: (v: { name: string; sortOrder: number }) =>
        vendorStoreApi.createCategory(storeId, v.name, v.sortOrder),
      onSuccess: invalidateMenu,
    }),
    updateCategory: useMutation({
      mutationFn: (v: { catId: string; body: Partial<{ name: string; sort_order: number }> }) =>
        vendorStoreApi.updateCategory(storeId, v.catId, v.body),
      onSuccess: invalidateMenu,
    }),
    deleteCategory: useMutation({
      mutationFn: (catId: string) => vendorStoreApi.deleteCategory(storeId, catId),
      onSuccess: invalidateMenu,
    }),
    createMenuItem: useMutation({
      mutationFn: (body: Parameters<typeof vendorStoreApi.createMenuItem>[1]) =>
        vendorStoreApi.createMenuItem(storeId, body),
      onSuccess: invalidateMenu,
    }),
    updateMenuItem: useMutation({
      mutationFn: (v: { itemId: string; body: Parameters<typeof vendorStoreApi.updateMenuItem>[2] }) =>
        vendorStoreApi.updateMenuItem(storeId, v.itemId, v.body),
      onSuccess: invalidateMenu,
    }),
    setMenuItemStatus: useMutation({
      mutationFn: (v: { itemId: string; status: string }) =>
        vendorStoreApi.setMenuItemStatus(storeId, v.itemId, v.status),
      onSuccess: invalidateMenu,
    }),
    deleteMenuItem: useMutation({
      mutationFn: (itemId: string) => vendorStoreApi.deleteMenuItem(storeId, itemId),
      onSuccess: invalidateMenu,
    }),
    createShipFee: useMutation({
      mutationFn: (body: Parameters<typeof vendorStoreApi.createShipFee>[1]) =>
        vendorStoreApi.createShipFee(storeId, body),
      onSuccess: invalidateShipFees,
    }),
    deleteShipFee: useMutation({
      mutationFn: (ruleId: string) => vendorStoreApi.deleteShipFee(storeId, ruleId),
      onSuccess: invalidateShipFees,
    }),
    uploadMenuItemImage: useMutation({
      mutationFn: (v: { itemId: string; file: File }) =>
        vendorStoreApi.uploadMenuItemImage(storeId, v.itemId, v.file),
      onSuccess: invalidateMenu,
    }),
    requestHoursChange: useMutation({
      mutationFn: (payload: Record<string, unknown>) =>
        vendorStoreApi.requestHoursChange(storeId, payload),
      onSuccess: () => qc.invalidateQueries({ queryKey: storeKeys.vendorHoursChange(storeId) }),
    }),
  };
}

export const useStoreShipFees = (storeId: string) =>
  useQuery({
    queryKey: storeKeys.shipFees(storeId),
    queryFn: () => vendorStoreApi.listShipFees(storeId),
    enabled: !!storeId,
  });

// Fetch current operating hours for a store.
export const useStoreHours = (storeId: string) =>
  useQuery({
    queryKey: storeKeys.storeHours(storeId),
    queryFn: () => browseStoreApi.hours(storeId),
    enabled: !!storeId,
  });

// Resolved unit ship fee for a specific location at any level (BUILDING | FLOOR | ROOM).
// Returns the query so callers can inspect isError (ErrLocationNotServed → HTTP 400).
export const useShipFee = (storeId: string, level: string, locationId: string) =>
  useQuery({
    queryKey: storeKeys.shipFee(storeId, level, locationId),
    queryFn: () => browseStoreApi.shipFee(storeId, level, locationId),
    enabled: !!storeId && !!level && !!locationId,
    // Do not retry on 4xx — "not served" is a definitive answer.
    retry: (failureCount, error) => {
      // axios errors expose response status
      const status = (error as { response?: { status?: number } }).response?.status;
      if (status && status >= 400 && status < 500) return false;
      return failureCount < 2;
    },
  });

// Vendor's own hours-change requests.
export const useVendorHoursChangeRequests = (storeId: string) =>
  useQuery({
    queryKey: storeKeys.vendorHoursChange(storeId),
    queryFn: () => vendorStoreApi.listHoursChangeRequests(storeId),
    enabled: !!storeId,
  });

// ---- Admin ----
export const useAdminStores = () =>
  useQuery({ queryKey: storeKeys.adminList(), queryFn: adminStoreApi.list });

export function useAdminStoreMutations() {
  const qc = useQueryClient();
  const invalidate = () => qc.invalidateQueries({ queryKey: storeKeys.adminList() });

  return {
    create: useMutation({
      mutationFn: (body: Parameters<typeof adminStoreApi.create>[0]) => adminStoreApi.create(body),
      onSuccess: invalidate,
    }),
    update: useMutation({
      mutationFn: (v: { id: string; body: Parameters<typeof adminStoreApi.update>[1] }) =>
        adminStoreApi.update(v.id, v.body),
      onSuccess: invalidate,
    }),
  };
}

export const useHoursChangeRequests = (storeId: string) =>
  useQuery({
    queryKey: storeKeys.hoursChange(storeId),
    queryFn: () => adminStoreApi.listHoursChangeRequests(storeId),
    enabled: !!storeId,
  });

export function useHoursChangeReviewMutations(storeId: string) {
  const qc = useQueryClient();
  const invalidate = () => qc.invalidateQueries({ queryKey: storeKeys.hoursChange(storeId) });

  return {
    approve: useMutation({
      mutationFn: (reqId: string) => adminStoreApi.approveHoursChange(storeId, reqId),
      onSuccess: invalidate,
    }),
    reject: useMutation({
      mutationFn: (reqId: string) => adminStoreApi.rejectHoursChange(storeId, reqId),
      onSuccess: invalidate,
    }),
  };
}

// ---- Option groups ----

export const useOptionGroups = (storeId: string) =>
  useQuery({
    queryKey: storeKeys.optionGroups(storeId),
    queryFn: () => vendorStoreApi.listOptionGroups(storeId),
    enabled: !!storeId,
  });

export const useOptions = (storeId: string, ogId: string) =>
  useQuery({
    queryKey: storeKeys.options(storeId, ogId),
    queryFn: () => vendorStoreApi.listOptions(storeId, ogId),
    enabled: !!storeId && !!ogId,
  });

export const useMenuItemOptionGroups = (storeId: string, itemId: string) =>
  useQuery({
    queryKey: storeKeys.menuItemOptionGroups(storeId, itemId),
    queryFn: () => vendorStoreApi.listMenuItemOptionGroups(storeId, itemId),
    enabled: !!storeId && !!itemId,
  });

export function useOptionGroupMutations(storeId: string) {
  const qc = useQueryClient();
  const invalidateGroups = () => qc.invalidateQueries({ queryKey: storeKeys.optionGroups(storeId) });

  return {
    createGroup: useMutation({
      mutationFn: (body: Parameters<typeof vendorStoreApi.createOptionGroup>[1]) =>
        vendorStoreApi.createOptionGroup(storeId, body),
      onSuccess: invalidateGroups,
    }),
    updateGroup: useMutation({
      mutationFn: (v: { ogId: string; body: Parameters<typeof vendorStoreApi.updateOptionGroup>[2] }) =>
        vendorStoreApi.updateOptionGroup(storeId, v.ogId, v.body),
      onSuccess: invalidateGroups,
    }),
    deleteGroup: useMutation({
      mutationFn: (ogId: string) => vendorStoreApi.deleteOptionGroup(storeId, ogId),
      onSuccess: invalidateGroups,
    }),
    createOption: useMutation({
      mutationFn: (v: { ogId: string; body: { name: string; extra_price: number } }) =>
        vendorStoreApi.createOption(storeId, v.ogId, v.body),
      onSuccess: (_data, v) => qc.invalidateQueries({ queryKey: storeKeys.options(storeId, v.ogId) }),
    }),
    updateOption: useMutation({
      mutationFn: (v: { ogId: string; optId: string; body: Partial<{ name: string; extra_price: number }> }) =>
        vendorStoreApi.updateOption(storeId, v.ogId, v.optId, v.body),
      onSuccess: (_data, v) => qc.invalidateQueries({ queryKey: storeKeys.options(storeId, v.ogId) }),
    }),
    deleteOption: useMutation({
      mutationFn: (v: { ogId: string; optId: string }) =>
        vendorStoreApi.deleteOption(storeId, v.ogId, v.optId),
      onSuccess: (_data, v) => qc.invalidateQueries({ queryKey: storeKeys.options(storeId, v.ogId) }),
    }),
  };
}

export function useMenuItemOptionGroupMutations(storeId: string, itemId: string) {
  const qc = useQueryClient();
  const invalidate = () => qc.invalidateQueries({ queryKey: storeKeys.menuItemOptionGroups(storeId, itemId) });

  return {
    attach: useMutation({
      mutationFn: (ogId: string) => vendorStoreApi.attachOptionGroup(storeId, itemId, ogId),
      onSuccess: invalidate,
    }),
    detach: useMutation({
      mutationFn: (ogId: string) => vendorStoreApi.detachOptionGroup(storeId, itemId, ogId),
      onSuccess: invalidate,
    }),
  };
}

// ---- Combos ----

export const useCombos = (storeId: string) =>
  useQuery({
    queryKey: storeKeys.combos(storeId),
    queryFn: () => vendorStoreApi.listCombos(storeId),
    enabled: !!storeId,
  });

export const useComboItems = (storeId: string, comboId: string) =>
  useQuery({
    queryKey: storeKeys.comboItems(storeId, comboId),
    queryFn: () => vendorStoreApi.listComboItems(storeId, comboId),
    enabled: !!storeId && !!comboId,
  });

export function useComboMutations(storeId: string) {
  const qc = useQueryClient();
  const invalidateCombos = () => qc.invalidateQueries({ queryKey: storeKeys.combos(storeId) });

  return {
    createCombo: useMutation({
      mutationFn: (body: { name: string; price: number }) =>
        vendorStoreApi.createCombo(storeId, body),
      onSuccess: invalidateCombos,
    }),
    updateCombo: useMutation({
      mutationFn: (v: { comboId: string; body: Partial<{ name: string; price: number }> }) =>
        vendorStoreApi.updateCombo(storeId, v.comboId, v.body),
      onSuccess: invalidateCombos,
    }),
    deleteCombo: useMutation({
      mutationFn: (comboId: string) => vendorStoreApi.deleteCombo(storeId, comboId),
      onSuccess: invalidateCombos,
    }),
    addComboItem: useMutation({
      mutationFn: (v: { comboId: string; menu_item_id: string; quantity: number }) =>
        vendorStoreApi.addComboItem(storeId, v.comboId, { menu_item_id: v.menu_item_id, quantity: v.quantity }),
      onSuccess: (_data, v) => qc.invalidateQueries({ queryKey: storeKeys.comboItems(storeId, v.comboId) }),
    }),
    removeComboItem: useMutation({
      mutationFn: (v: { comboId: string; menuItemId: string }) =>
        vendorStoreApi.removeComboItem(storeId, v.comboId, v.menuItemId),
      onSuccess: (_data, v) => qc.invalidateQueries({ queryKey: storeKeys.comboItems(storeId, v.comboId) }),
    }),
  };
}

