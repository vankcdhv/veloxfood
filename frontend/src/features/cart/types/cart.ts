// Mirrors backend order-service cart entities.
// All entity fields are PascalCase to match Go's default JSON encoding.
// Request bodies use snake_case.

export interface CartItem {
  MenuItemID: string;
  NameSnapshot: string;
  PriceSnapshot: number;
  Qty: number;
  // These may be empty for stores without slot-based ordering.
  CutoffID?: string;
  Date?: string;
}

export interface Cart {
  StoreID: string;
  StoreName?: string;
  Items: CartItem[];
}

// PUT /api/v1/me/cart/items/:menuItemId
export interface UpdateCartItemBody {
  qty: number;
  cutoff_id?: string;
  date?: string;
}
