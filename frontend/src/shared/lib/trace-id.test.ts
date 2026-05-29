import { describe, expect, it } from 'vitest';
import { generateTraceId, TRACE_ID_HEADER } from './trace-id';

describe('trace-id', () => {
  it('exports canonical header name', () => {
    expect(TRACE_ID_HEADER).toBe('X-Trace-Id');
  });

  it('generates RFC4122 v4 UUID', () => {
    const id = generateTraceId();
    expect(id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
  });

  it('returns different ids on each call', () => {
    expect(generateTraceId()).not.toBe(generateTraceId());
  });
});
