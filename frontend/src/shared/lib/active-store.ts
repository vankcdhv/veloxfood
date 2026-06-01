'use client';

import { useSyncExternalStore } from 'react';

// Tracks which store the customer is currently ordering from. The cart is
// per-store on the backend (every cart endpoint needs store_id), so we persist
// the active store client-side: set it when an item is added (or a store page
// is opened) and read it back on the standalone /cart + header cart icon.

const KEY = 'velox.activeStoreId';
const EVENT = 'velox:active-store-changed';

export function getActiveStoreId(): string | null {
  if (typeof window === 'undefined') return null;
  return window.localStorage.getItem(KEY);
}

export function setActiveStoreId(id: string): void {
  if (typeof window === 'undefined') return;
  if (window.localStorage.getItem(KEY) === id) return;
  window.localStorage.setItem(KEY, id);
  window.dispatchEvent(new Event(EVENT));
}

export function clearActiveStoreId(): void {
  if (typeof window === 'undefined') return;
  window.localStorage.removeItem(KEY);
  window.dispatchEvent(new Event(EVENT));
}

function subscribe(cb: () => void): () => void {
  if (typeof window === 'undefined') return () => {};
  window.addEventListener(EVENT, cb);
  window.addEventListener('storage', cb); // cross-tab
  return () => {
    window.removeEventListener(EVENT, cb);
    window.removeEventListener('storage', cb);
  };
}

/** Reactive read of the active store id; re-renders on change (this or other tabs). */
export function useActiveStoreId(): string | null {
  return useSyncExternalStore(subscribe, getActiveStoreId, () => null);
}
