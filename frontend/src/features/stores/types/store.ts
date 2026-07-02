// Mirrors backend store-service entities.
// All fields are PascalCase to match Go's default JSON encoding.

export type SaleStatus = 'OPEN' | 'CLOSED_TODAY' | 'PAUSED';
export type MenuItemStatus = 'on' | 'off';
export type ShipFeeScope = 'building' | 'floor' | 'room';

export interface Store {
  ID: string;
  OwnerUserID: string;
  OwnerUserName?: string; // Enriched by store-service via gRPC user lookup
  VendorID: string;
  Name: string;
  BusinessType: string;
  Address: string;
  Phone: string;
  SaleStatus: SaleStatus;
  PickupEnabled: boolean;
  AvatarURL?: string;
  CreatedAt: string;
  // Prep-time + open-now fields returned by browse/detail endpoints.
  PrepMinutes?: number;
  OpenNow?: boolean;
  OpenTimeToday?: string;  // "HH:MM" or ""
  CloseTimeToday?: string; // "HH:MM" or ""
}

export interface Category {
  ID: string;
  StoreID: string;
  Name: string;
  SortOrder: number;
}

export interface MenuItem {
  ID: string;
  StoreID: string;
  CategoryID: string;
  Name: string;
  Description: string;
  Price: number;
  ImageURL: string;
  Status: MenuItemStatus;
  // Backend stores tags as a comma-separated string (e.g. "bestseller,new").
  Tags: string;
}

export interface ShipFeeRule {
  ID: string;
  StoreID: string;
  Scope: ShipFeeScope;
  RefID: string;
  UnitFee: number;
}

// Menu response groups items by category.
export interface MenuCategory {
  Category: Category;
  Items: MenuItem[];
}

// Resolved ship fee for a specific room. The browse endpoint returns the value
// under a snake_case key (gin.H), unlike the PascalCase entity responses.
export interface ShipFeeResult {
  unit_ship_fee: number;
}

// Operating hours per weekday (0=Sunday … 6=Saturday). Format: "HH:MM".
export interface OperatingHour {
  ID: string;
  StoreID: string;
  Weekday: number;
  OpenTime: string;
  CloseTime: string;
}


export interface StoreHoursResponse {
  operating_hours: OperatingHour[];
}

// Structured payload inside an hours-change request.
export interface HoursChangePayload {
  operating_hours: Array<{ weekday: number; open_time: string; close_time: string }>;
  note?: string;
}

// Admin: hours-change request stub (minimal fields returned by list/review).
export interface HoursChangeRequest {
  ID: string;
  StoreID: string;
  Status: string;
  // Payload is stored as the structured hoursChangePayload JSON (not freeform).
  Payload: HoursChangePayload;
  CreatedAt: string;
}

// ---- Option groups & options ----

export interface OptionGroup {
  ID: string;
  StoreID: string;
  Name: string;
  MinSelect: number;
  MaxSelect: number;
  Required: boolean;
}

export interface Option {
  ID: string;
  OptionGroupID: string;
  Name: string;
  ExtraPrice: number;
}

// ---- Combos & combo items ----

export interface Combo {
  ID: string;
  StoreID: string;
  Name: string;
  Price: number;
}

export interface ComboItem {
  ID: string;
  ComboID: string;
  MenuItemID: string;
  Quantity: number;
}

// ---- Search ----

// Returned by GET /api/v1/menu-items/search. Fields are PascalCase to match Go JSON.
export interface MenuItemSearchResult {
  ID: string;
  Name: string;
  Price: number;
  ImageURL: string;
  Description: string;
  StoreID: string;
  StoreName: string;
  SaleStatus: SaleStatus;
  // Store is currently open (operating hours + sale status), server-computed.
  OpenNow: boolean;
}

export type MenuItemSearchSort = 'relevance' | 'price_asc' | 'price_desc';

// Server-side narrowing for menu-item search. Zero/empty values are omitted.
export interface MenuItemSearchFilters {
  sort?: MenuItemSearchSort;
  priceMin?: number;
  priceMax?: number;
  storeId?: string;
}

