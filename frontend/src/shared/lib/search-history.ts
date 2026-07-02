// Recent-search persistence (localStorage). Most-recent first, deduplicated
// case-insensitively, capped at 10 entries. All functions no-op safely when
// localStorage is unavailable (SSR, private mode).

const KEY = 'veloxfood.search-history';
const MAX_ENTRIES = 10;

export function getSearchHistory(): string[] {
  try {
    const raw = localStorage.getItem(KEY);
    const parsed: unknown = raw ? JSON.parse(raw) : [];
    return Array.isArray(parsed) ? parsed.filter((s): s is string => typeof s === 'string') : [];
  } catch {
    return [];
  }
}

export function addSearchHistory(term: string): string[] {
  const t = term.trim();
  if (!t) return getSearchHistory();
  const next = [t, ...getSearchHistory().filter((s) => s.toLowerCase() !== t.toLowerCase())].slice(
    0,
    MAX_ENTRIES,
  );
  try {
    localStorage.setItem(KEY, JSON.stringify(next));
  } catch {
    // quota/unavailable — history just isn't persisted
  }
  return next;
}

export function removeSearchHistory(term: string): string[] {
  const next = getSearchHistory().filter((s) => s !== term);
  try {
    localStorage.setItem(KEY, JSON.stringify(next));
  } catch {
    // ignore
  }
  return next;
}

export function clearSearchHistory(): void {
  try {
    localStorage.removeItem(KEY);
  } catch {
    // ignore
  }
}
