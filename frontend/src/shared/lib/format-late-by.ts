// formatLateBy renders a late-by duration (in minutes) compactly in Vietnamese:
//   45    → "45'"
//   185   → "3h 5'"
//   2925  → "2 ngày 45'"
// so large values (e.g. an order delivered days later) stay readable instead of
// showing a giant raw minute count.
export function formatLateBy(minutes: number): string {
  const total = Math.max(0, Math.round(minutes));
  if (total < 60) return `${total}'`;

  if (total < 1440) {
    const h = Math.floor(total / 60);
    const m = total % 60;
    return m ? `${h}h ${m}'` : `${h}h`;
  }

  const days = Math.floor(total / 1440);
  const h = Math.floor((total % 1440) / 60);
  const m = total % 60;
  let out = `${days} ngày`;
  if (h) out += ` ${h}h`;
  if (m) out += ` ${m}'`;
  return out;
}
