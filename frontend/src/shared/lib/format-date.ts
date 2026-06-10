// Vietnamese date/time formatting helpers. Accept ISO strings, epoch ms, or Date.
// Return '—' for empty/invalid input so callers never render "Invalid Date".

function toDate(input: string | number | Date | null | undefined): Date | null {
  if (input === null || input === undefined || input === '') return null;
  const d = input instanceof Date ? input : new Date(input);
  if (Number.isNaN(d.getTime())) return null;
  // Treat Go's zero time ("0001-01-01…") as empty so it renders "—", not "01/01/1".
  if (d.getFullYear() < 2000) return null;
  return d;
}

// "01/06/2026"
export function formatDate(input: string | number | Date | null | undefined): string {
  const d = toDate(input);
  if (!d) return '—';
  return `${String(d.getDate()).padStart(2, '0')}/${String(d.getMonth() + 1).padStart(2, '0')}/${d.getFullYear()}`;
}

// "01/06/2026 17:40"
export function formatDateTime(input: string | number | Date | null | undefined): string {
  const d = toDate(input);
  if (!d) return '—';
  return `${formatDate(d)} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}
