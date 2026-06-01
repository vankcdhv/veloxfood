import type { ShipCutoff } from '../types/store';

// Local YYYY-MM-DD for a Date (matches the date-picker value format).
function isoDateOf(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

// An order for a session must be placed before its deadline = CutoffTime − LeadMinutes
// (e.g. ca 22:00 with 30' lead closes at 21:30). A cutoff is "closed" only for the
// current calendar day once that deadline has passed; future dates are always open.
export function isCutoffClosed(cutoff: ShipCutoff, isoDate: string, now: Date = new Date()): boolean {
  if (isoDate !== isoDateOf(now)) return false;
  const [h, m] = cutoff.CutoffTime.split(':').map(Number);
  const deadline = new Date(now);
  deadline.setHours(h, m - (cutoff.LeadMinutes ?? 0), 0, 0);
  return now.getTime() >= deadline.getTime();
}

// Pick the effective session: the user's choice if it's still open, otherwise the
// first open cutoff for the date. Returns '' when every cutoff is closed.
export function resolveActiveCutoff(
  cutoffs: ShipCutoff[],
  selected: string,
  isoDate: string,
  now: Date = new Date(),
): string {
  const open = cutoffs.filter((c) => !isCutoffClosed(c, isoDate, now));
  if (open.some((c) => c.ID === selected)) return selected;
  return open[0]?.ID ?? '';
}
