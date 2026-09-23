import { describe, expect, it } from 'vitest';
import { centsToDollarsInput, dollarsToCents, formatDate, formatMoney, formatMoneyWhole, parseDate } from './format';

describe('dollarsToCents', () => {
  it('converts typed amounts to integer cents', () => {
    expect(dollarsToCents('1500')).toBe(150000);
    expect(dollarsToCents('19.99')).toBe(1999);
    expect(dollarsToCents('$1,234.50')).toBe(123450);
  });

  it('does not drift on floating-point-hostile values', () => {
    expect(dollarsToCents('0.1')).toBe(10);
    expect(dollarsToCents('1.005')).toBe(101);
    expect(dollarsToCents('4.35')).toBe(435);
  });

  it('returns undefined for blank or invalid input', () => {
    expect(dollarsToCents('')).toBeUndefined();
    expect(dollarsToCents('   ')).toBeUndefined();
    expect(dollarsToCents('abc')).toBeUndefined();
  });
});

describe('centsToDollarsInput', () => {
  it('round-trips with dollarsToCents', () => {
    expect(centsToDollarsInput(123450)).toBe('1234.50');
    expect(dollarsToCents(centsToDollarsInput(999))).toBe(999);
  });
});

describe('money formatting', () => {
  it('formats cents as dollars', () => {
    expect(formatMoney(123456)).toBe('$1,234.56');
    expect(formatMoney(0)).toBe('$0.00');
    expect(formatMoneyWhole(123456)).toBe('$1,235');
  });
});

describe('date parsing', () => {
  it('reads a bare date as a LOCAL date, not UTC midnight', () => {
    const d = parseDate('2026-09-01');
    expect(d?.getFullYear()).toBe(2026);
    expect(d?.getMonth()).toBe(8);
    expect(d?.getDate()).toBe(1); // new Date('2026-09-01') would be Aug 31 in negative-offset zones
  });

  it('formats dates and tolerates blanks', () => {
    expect(formatDate('2026-09-01')).toBe('Sep 1, 2026');
    expect(formatDate('')).toBe('—');
    expect(parseDate('not a date')).toBeNull();
  });
});
