import { tokens } from '@/app/tokens';

interface RentFlowMarkProps {
  size?: number;
  /** 'white' is for dark/photo backgrounds (see AuthLayout); 'color' is the default ink+green mark. */
  variant?: 'color' | 'white';
}

/**
 * The RentFlow symbol — a roofline over a ledger bar ("Roofline Ledger"
 * mark). Kept as inline paths (not an <img>) so it stays crisp at the
 * 20-32px sizes used across the nav and auth screens. Only the short
 * bar carries the accent color; never recolor the roof stroke.
 */
export function RentFlowMark({ size = 32, variant = 'color' }: RentFlowMarkProps) {
  const roof = variant === 'white' ? '#FFFFFF' : tokens.brand.ink;
  const accent = variant === 'white' ? tokens.brand.mint : tokens.brand.green;

  return (
    <svg width={size} height={size} viewBox="0 0 64 64" fill="none" aria-hidden="true">
      <g stroke={roof} strokeWidth={7} strokeLinecap="round" strokeLinejoin="round">
        <path d="M7 27 L32 7 L57 27" />
        <path d="M16 41 H48" />
      </g>
      <path d="M16 54 H36" stroke={accent} strokeWidth={7} strokeLinecap="round" />
    </svg>
  );
}
