import { describe, expect, it } from 'vitest';
import { createPropertySchema } from './propertySchema';

describe('createPropertySchema', () => {
  it('accepts valid input', () => {
    const result = createPropertySchema.safeParse({ address: '123 Main St', unitCount: 4, status: 'active' });
    expect(result.success).toBe(true);
  });

  it('rejects an address shorter than 3 characters', () => {
    const result = createPropertySchema.safeParse({ address: 'ab', unitCount: 4, status: 'active' });
    expect(result.success).toBe(false);
  });

  it('rejects a whitespace-only address', () => {
    const result = createPropertySchema.safeParse({ address: '      ', unitCount: 4, status: 'active' });
    expect(result.success).toBe(false);
  });

  it('trims address whitespace', () => {
    const result = createPropertySchema.safeParse({ address: '  123 Main St  ', unitCount: 4, status: 'active' });
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.address).toBe('123 Main St');
    }
  });

  it('rejects a unit count below 1', () => {
    const result = createPropertySchema.safeParse({ address: '123 Main St', unitCount: 0, status: 'active' });
    expect(result.success).toBe(false);
  });

  it('rejects a non-integer unit count', () => {
    const result = createPropertySchema.safeParse({ address: '123 Main St', unitCount: 1.5, status: 'active' });
    expect(result.success).toBe(false);
  });

  it('rejects an unknown status', () => {
    const result = createPropertySchema.safeParse({ address: '123 Main St', unitCount: 4, status: 'condemned' });
    expect(result.success).toBe(false);
  });
});
