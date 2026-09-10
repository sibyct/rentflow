import { describe, expect, it } from 'vitest';
import { loginSchema, registerSchema } from './authSchema';

describe('loginSchema', () => {
  it('accepts valid input', () => {
    const result = loginSchema.safeParse({ email: 'owner@example.com', password: 'anything' });
    expect(result.success).toBe(true);
  });

  it('rejects a missing email', () => {
    const result = loginSchema.safeParse({ email: '', password: 'anything' });
    expect(result.success).toBe(false);
  });

  it('rejects a malformed email', () => {
    const result = loginSchema.safeParse({ email: 'not-an-email', password: 'anything' });
    expect(result.success).toBe(false);
  });

  it('rejects a missing password', () => {
    const result = loginSchema.safeParse({ email: 'owner@example.com', password: '' });
    expect(result.success).toBe(false);
  });

  it('trims email whitespace', () => {
    const result = loginSchema.safeParse({ email: '  owner@example.com  ', password: 'anything' });
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.email).toBe('owner@example.com');
    }
  });
});

describe('registerSchema', () => {
  it('accepts valid input', () => {
    const result = registerSchema.safeParse({ email: 'owner@example.com', password: 'hunter22222' });
    expect(result.success).toBe(true);
  });

  it('rejects a password shorter than 8 characters', () => {
    const result = registerSchema.safeParse({ email: 'owner@example.com', password: 'short' });
    expect(result.success).toBe(false);
  });

  it('rejects a password longer than 72 characters', () => {
    const result = registerSchema.safeParse({ email: 'owner@example.com', password: 'a'.repeat(73) });
    expect(result.success).toBe(false);
  });
});
