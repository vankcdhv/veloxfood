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
  CreatedAt: string;
  ShipCutoffs?: ShipCutoff[]; // Ca giao, attached by the store list/detail endpoints
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

// Ship cut-off time slot. CutoffTime: "HH:MM".
export interface ShipCutoff {
  ID: string;
  StoreID: string;
  CutoffTime: string;
  LeadMinutes: number;
}

export interface StoreHoursResponse {
  operating_hours: OperatingHour[];
  ship_cutoffs: ShipCutoff[];
}

// Remaining slot quota for one (menu item, cutoff) on a given date.
export interface StoreSlotQuota {
  MenuItemID: string;
  CutoffID: string;
  Quota: number;
  SoldCount: number;
  Remaining: number;
}

// GET /stores/:id/slots?date= — cutoffs + per-item remaining quota for the date.
export interface StoreSlotsResponse {
  cutoffs: ShipCutoff[];
  quotas: StoreSlotQuota[];
}

// Structured payload inside an hours-change request.
export interface HoursChangePayload {
  operating_hours: Array<{ weekday: number; open_time: string; close_time: string }>;
  ship_cutoffs: Array<{ cutoff_time: string; lead_minutes: number }>;
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

// ---- Slot quota ----

export interface SlotQuota {
  ID: string;
  MenuItemID: string;
  Date: string;        // "YYYY-MM-DD"
  CutoffID: string;
  Quota: number;
  SoldCount: number;
}
