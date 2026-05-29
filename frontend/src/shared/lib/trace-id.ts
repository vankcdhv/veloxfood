// Match backend pkg/trace: X-Trace-Id header, UUID v4. Backend canonical name
// is "X-Trace-Id" (mixed-case) — see project/backend/pkg/trace/trace.go.
export const TRACE_ID_HEADER = 'X-Trace-Id';

export function generateTraceId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  // Fallback for environments without WebCrypto (older Node in tests).
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}
