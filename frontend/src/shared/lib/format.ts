// Shared money/date formatting for the accounting screens. Money is always
// integer cents on the wire (see backend/internal/domain/accounting.go) and
// only becomes a dollar string here, at the display edge.

const money = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' });
const moneyWhole = new Intl.NumberFormat('en-US', {
  style: 'currency',
  currency: 'USD',
  maximumFractionDigits: 0,
});
const dateFormatter = new Intl.DateTimeFormat('en-US', { dateStyle: 'medium' });
const monthFormatter = new Intl.DateTimeFormat('en-US', { month: 'short', year: 'numeric' });

/** 123456 → "$1,234.56". */
export function formatMoney(cents: number): string {
  return money.format(cents / 100);
}

/** 123456 → "$1,235" — for KPI cards and chart axes where cents are noise. */
export function formatMoneyWhole(cents: number): string {
  return moneyWhole.format(cents / 100);
}

/**
 * Parses an ISO date or timestamp. A bare "YYYY-MM-DD" is read as a LOCAL
 * date — `new Date('2026-09-01')` is UTC midnight and renders as Aug 31 in
 * any negative-offset time zone, which for a rent due date is a real bug.
 */
export function parseDate(iso: string): Date | null {
  if (!iso) return null;
  const dateOnly = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso);
  const d = dateOnly
    ? new Date(Number(dateOnly[1]), Number(dateOnly[2]) - 1, Number(dateOnly[3]))
    : new Date(iso);
  return Number.isNaN(d.getTime()) ? null : d;
}

export function formatDate(iso: string): string {
  const d = parseDate(iso);
  return d ? dateFormatter.format(d) : iso || '—';
}

/** "2026-09-01" → "Sep 2026". */
export function formatMonth(iso: string): string {
  const d = parseDate(iso);
  return d ? monthFormatter.format(d) : iso || '—';
}

/** Local today as "YYYY-MM-DD" (what a native date input expects). */
export function todayIso(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

/** Local current month as "YYYY-MM" (what a native month input expects). */
export function currentMonthIso(): string {
  return todayIso().slice(0, 7);
}

/**
 * "1,234.50" → 123450. Undefined for blank/invalid input. Parsed from the
 * decimal string, not via `Number(x) * 100`: in floating point 1.005 * 100
 * is 100.49999…, which would silently round a typed "$1.005" down.
 * Anything past the cent rounds half up.
 */
export function dollarsToCents(input: string): number | undefined {
  const cleaned = input.replace(/[$,\s]/g, '');
  const m = /^(-?)(\d*)(?:\.(\d*))?$/.exec(cleaned);
  if (!m) return undefined;
  const whole = m[2] ?? '';
  const frac = m[3] ?? '';
  if (whole === '' && frac === '') return undefined;

  const padded = frac.padEnd(3, '0');
  let cents = Number(whole || '0') * 100 + Number(padded.slice(0, 2));
  if (Number(padded[2]) >= 5) cents += 1;
  return m[1] === '-' ? -cents : cents;
}

/** 123450 → "1234.50" (a form field's editable value, no symbols). */
export function centsToDollarsInput(cents: number): string {
  return (cents / 100).toFixed(2);
}
